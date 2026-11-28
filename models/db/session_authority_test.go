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

func TestSessionAuthorityAfterLock(t *testing.T) {
	source := os.Getenv("BDSPRO_SESSION_AUTHORITY_TEST_DATABASE_URL")
	if source == "" {
		t.Skip("requires disposable migrated BDSPRO_SESSION_AUTHORITY_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
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
	q := sqlc.New(setup)
	personID, err := q.CreateParty(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.CreatePerson(ctx, personID); err != nil {
		t.Fatal(err)
	}
	account, err := q.CreateAccountWithoutEmail(ctx, personID)
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
			{"DELETE FROM sessions WHERE account_id=$1", account.ID},
			{"DELETE FROM accounts WHERE id=$1", account.ID},
			{"DELETE FROM persons WHERE party_id=$1", personID},
			{"DELETE FROM parties WHERE id=$1", personID},
		} {
			if _, err := locker.Exec(cleanup, statement.sql, statement.id); err != nil {
				t.Error("fixture cleanup failed", err)
			}
		}
	}()

	// Each invocation transfers exclusive use of the locker transaction to
	// a goroutine, proves the other connection is waiting, then releases it.
	releaseAfterWait := func(block pgx.Tx, deadline time.Time) <-chan error {
		result := make(chan error, 1)
		lockerPID := int32(locker.PgConn().PID())
		waiterPID := int32(waiter.PgConn().PID())
		go func() {
			result <- func() error {
				defer block.Rollback(context.Background())
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for {
					var waiting bool
					if err := block.QueryRow(ctx, "SELECT $2::integer=ANY(pg_blocking_pids($1::integer))", waiterPID, lockerPID).Scan(&waiting); err != nil {
						return err
					}
					if waiting {
						break
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-ticker.C:
					}
				}
				if !deadline.IsZero() {
					if _, err := block.Exec(ctx, "SELECT pg_sleep(GREATEST(0,EXTRACT(EPOCH FROM ($1::timestamptz-clock_timestamp()))::double precision+0.05))", deadline); err != nil {
						return err
					}
				}
				return block.Commit(ctx)
			}()
		}()
		return result
	}

	sessionID := uuid.New()
	var deadline time.Time
	if err := locker.QueryRow(ctx, "INSERT INTO sessions(id,account_id,refresh_token,user_agent,client_ip,expires_at) VALUES ($1,$2,'fixture','fixture','127.0.0.1',clock_timestamp()+interval '5 seconds') RETURNING expires_at", sessionID, account.ID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	block, err := locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer block.Rollback(context.Background())
	if _, err := block.Exec(ctx, "SELECT id FROM sessions WHERE id=$1 FOR UPDATE", sessionID); err != nil {
		t.Fatal(err)
	}
	tx, err := waiter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	q = sqlc.New(tx)
	if _, err := q.LockSessionRevocationAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	released := releaseAfterWait(block, deadline)
	_, lockErr := q.LockActingSession(ctx, sqlc.LockActingSessionParams{SessionID: sessionID, AccountID: account.ID})
	if err := <-released; err != nil {
		t.Fatal("session lock wait not verified", err)
	}
	if !errors.Is(lockErr, pgx.ErrNoRows) {
		t.Fatal("session expired while waiting but was admitted", lockErr)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	t.Log("PASS: session rejected after confirmed lock wait crossed expiry")

	// Restore only the disposable session to test recent-auth consumption.
	if _, err := locker.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()+interval '10 minutes' WHERE id=$1", sessionID); err != nil {
		t.Fatal(err)
	}
	stale := sha256.Sum256([]byte(uuid.NewString()))
	if err := locker.QueryRow(ctx, "INSERT INTO auth_proofs(digest,account_id,session_id,action,method,expires_at) VALUES ($1,$2,$3,'sessions.revoke_all','password',clock_timestamp()+interval '5 seconds') RETURNING expires_at", stale[:], account.ID, sessionID).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	block, err = locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer block.Rollback(context.Background())
	if _, err := block.Exec(ctx, "SELECT digest FROM auth_proofs WHERE digest=$1 FOR UPDATE", stale[:]); err != nil {
		t.Fatal(err)
	}
	tx, err = waiter.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	q = sqlc.New(tx)
	if _, err := q.LockSessionRevocationAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := q.LockActingSession(ctx, sqlc.LockActingSessionParams{SessionID: sessionID, AccountID: account.ID}); err != nil {
		t.Fatal(err)
	}
	binding := sqlc.ConsumeRecentAuthProofParams{Digest: stale[:], AccountID: account.ID, SessionID: sessionID, Action: "sessions.revoke_all"}
	released = releaseAfterWait(block, deadline)
	_, consumeErr := q.ConsumeRecentAuthProof(ctx, binding)
	if err := <-released; err != nil {
		t.Fatal("proof lock wait not verified", err)
	}
	if !errors.Is(consumeErr, pgx.ErrNoRows) {
		t.Fatal("proof expired while waiting but was consumed", consumeErr)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := locker.QueryRow(ctx, "SELECT count(*) FROM auth_proofs WHERE digest=$1", stale[:]).Scan(&remaining); err != nil || remaining != 1 {
		t.Fatal("expired proof was deleted", err)
	}
	t.Log("PASS: recent-auth proof rejected after confirmed lock wait crossed expiry")

	valid := sha256.Sum256([]byte(uuid.NewString()))
	if err := sqlc.New(locker).CreateRecentAuthProof(ctx, sqlc.CreateRecentAuthProofParams{
		Digest: valid[:], AccountID: account.ID, SessionID: sessionID,
		Action: "sessions.revoke_all", Method: "password", TtlMinutes: 10,
	}); err != nil {
		t.Fatal(err)
	}
	binding.Digest = valid[:]
	wrong := binding
	wrong.Action = "email.link"
	if _, err := sqlc.New(waiter).ConsumeRecentAuthProof(ctx, wrong); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("wrong action consumed proof", err)
	}
	block, err = locker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer block.Rollback(context.Background())
	if row, err := sqlc.New(block).ConsumeRecentAuthProof(ctx, binding); err != nil || row.Method != "password" {
		t.Fatal("valid proof not consumed", err)
	}
	// Exercise the generated SQL directly here: the service normally
	// serializes the two consumers even earlier on the account lock.
	released = releaseAfterWait(block, time.Time{})
	_, consumeErr = sqlc.New(waiter).ConsumeRecentAuthProof(ctx, binding)
	if err := <-released; err != nil {
		t.Fatal("concurrent consumption wait not verified", err)
	}
	if !errors.Is(consumeErr, pgx.ErrNoRows) {
		t.Fatal("second consumer accepted same proof", consumeErr)
	}
	t.Log("PASS: one proof consumer, concurrent replay rejected")
}
