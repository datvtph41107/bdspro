package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Use two independent connections to demonstrate the difference between
// transaction-start time and expiry at consumption after a row-lock wait.
func TestPasswordResetLockExpiry(t *testing.T) {
	source := os.Getenv("BDSPRO_PASSWORD_RESET_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_PASSWORD_RESET_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	locker, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer locker.Close(context.Background())
	waiter, err := pgx.Connect(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	defer waiter.Close(context.Background())

	setup, err := locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Rollback(context.Background())
	fixture := sqlc.New(setup)
	personID, err := fixture.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.CreatePerson(ctx, personID); err != nil {
		t.Fatal(err)
	}
	account, err := fixture.CreateAccountWithoutEmail(ctx, personID)
	if err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, statement := range []struct {
			sql string
			id  int64
		}{
			{"DELETE FROM accounts WHERE id=$1", account.ID},
			{"DELETE FROM persons WHERE party_id=$1", personID},
			{"DELETE FROM parties WHERE id=$1", personID},
		} {
			if _, err := locker.Exec(cleanup, statement.sql, statement.id); err != nil {
				t.Error("fixture cleanup failed", err)
			}
		}
	}()

	stale := sha256.Sum256([]byte(uuid.NewString()))
	var deadline time.Time
	if err := locker.QueryRow(ctx, "INSERT INTO password_reset_tokens(token_hash,account_id,expires_at) VALUES ($1,$2,clock_timestamp()+interval '5 seconds') RETURNING expires_at", stale[:], account.ID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	block, err := locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer block.Rollback(context.Background())
	if _, err := sqlc.New(block).LockPasswordResetOwner(ctx, stale[:]); err != nil {
		t.Fatal(err)
	}
	tx, err := waiter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	q := sqlc.New(tx)
	owner, err := q.ReadActivePasswordResetOwner(ctx, stale[:])
	if err != nil || owner != account.ID {
		t.Fatal("fixture must be active before the lock wait", err)
	}
	if _, err := q.LockSessionRevocationAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	lockerPID := int32(locker.PgConn().PID())
	waiterPID := int32(waiter.PgConn().PID())
	done := make(chan error, 1)
	go func() {
		done <- func() error {
			defer block.Rollback(context.Background())
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				var blocked bool
				if err := block.QueryRow(ctx, "SELECT $2::integer = ANY(pg_blocking_pids($1::integer))", waiterPID, lockerPID).Scan(&blocked); err != nil {
					return err
				}
				if blocked {
					break
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
			// Release only after the database's own clock crosses the deadline.
			if _, err := block.Exec(ctx, "SELECT pg_sleep(GREATEST(0, EXTRACT(EPOCH FROM ($1::timestamptz-clock_timestamp()))::double precision+0.05))", deadline); err != nil {
				return err
			}
			return block.Commit(ctx)
		}()
	}()
	owner, lockErr := q.LockPasswordResetOwner(ctx, stale[:])
	releaseErr := <-done
	if lockErr != nil || releaseErr != nil || owner != account.ID {
		t.Fatal("expected proof row lock after a confirmed wait", lockErr, releaseErr)
	}
	var oldPredicate, expired bool
	if err := tx.QueryRow(ctx, "SELECT expires_at>now(), expires_at<=clock_timestamp() FROM password_reset_tokens WHERE token_hash=$1", stale[:]).Scan(&oldPredicate, &expired); err != nil {
		t.Fatal(err)
	}
	if !oldPredicate || !expired {
		t.Fatal("fixture did not demonstrate stale transaction time")
	}
	if _, err := q.ConsumePasswordResetToken(ctx, stale[:]); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expired proof was accepted after lock wait", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := locker.QueryRow(ctx, "SELECT count(*) FROM password_reset_tokens WHERE token_hash=$1", stale[:]).Scan(&remaining); err != nil || remaining != 1 {
		t.Fatal("expired proof was unexpectedly deleted", err)
	}
	t.Log("PASS: confirmed row-lock wait; now() still admitted expired proof, consume rejected it")

	valid := sha256.Sum256([]byte(uuid.NewString()))
	if err := sqlc.New(locker).CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		TokenHash: valid[:], AccountID: account.ID, TtlMinutes: 15,
	}); err != nil {
		t.Fatal(err)
	}
	live, err := waiter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Rollback(context.Background())
	q = sqlc.New(live)
	if _, err := q.LockSessionRevocationAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	if owner, err := q.LockPasswordResetOwner(ctx, valid[:]); err != nil || owner != account.ID {
		t.Fatal("valid proof lock failed", err)
	}
	if owner, err := q.ConsumePasswordResetToken(ctx, valid[:]); err != nil || owner != account.ID {
		t.Fatal("valid proof consume failed", err)
	}
	if err := live.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlc.New(locker).LockPasswordResetOwner(ctx, valid[:]); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("consumed proof could be replayed", err)
	}
	t.Log("PASS: valid proof consumed exactly once")
}
