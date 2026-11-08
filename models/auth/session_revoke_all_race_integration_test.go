package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRevokeOnConn struct {
	SessionStore
	conn *pgxpool.Conn
}

func (s sessionRevokeOnConn) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func g12cStore(t *testing.T) (context.Context, *pgxpool.Pool, *db.SQLStore) {
	t.Helper()
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable migrated PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing writes without disposable test mode")
	}
	config, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if config.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing non-test PostgreSQL")
	}
	config.MaxConns = max(config.MaxConns, 6)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, db.NewStore(pool)
}

func g12cSessions(t *testing.T, ctx context.Context, store *db.SQLStore) (int64, uuid.UUID, uuid.UUID) {
	t.Helper()
	account, err := db.WithTxResult(ctx, store, func(q *db.Queries) (db.Account, error) {
		personID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return db.Account{}, err
		}
		account, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: personID, Email: uuid.NewString() + "@example.test",
		})
		if err != nil {
			return db.Account{}, err
		}
		return account, q.CreateAccountSettings(ctx, account.ID)
	})
	if err != nil {
		t.Fatal(err)
	}
	newSession := func() uuid.UUID {
		id := uuid.New()
		if _, err := store.CreateSession(ctx, db.CreateSessionParams{
			ID: id, AccountID: account.ID,
			RefreshToken: "sha256:" + strings.Repeat("a", 64) + uuid.NewString(),
			UserAgent:    "g1-2c-revoke", ClientIp: "203.0.113.23",
			ExpiresAt: time.Now().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return account.ID, newSession(), newSession()
}

func TestRevokeAllCannotEscapeConcurrentActingSessionRevoke(t *testing.T) {
	ctx, pool, store := g12cStore(t)
	accountID, actingID, otherID := g12cSessions(t, ctx, store)

	revoker, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer revoker.Release()
	writer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Release()
	revokerPID := int32(revoker.Conn().PgConn().PID())
	writerPID := int32(writer.Conn().PgConn().PID())

	tx, err := revoker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", actingID); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		result <- RevokeAllSessions(ctx, sessionRevokeOnConn{SessionStore: store, conn: writer}, accountID, actingID)
	}()

	// A real pg_blocking_pids edge proves the candidate's authority check
	// waits on the concurrent revocation, not a timing-dependent sleep.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	blocked := false
	for !blocked {
		select {
		case err := <-result:
			t.Fatalf("account-wide revoke finished before acting session revoke: %v", err)
		case <-tick.C:
			if err := pool.QueryRow(ctx,
				"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
				revokerPID, writerPID,
			).Scan(&blocked); err != nil {
				t.Fatal(err)
			}
		case <-timeout.C:
			t.Fatal("revoke-all never waited for acting session authority")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, ErrSessionUnavailable) {
			t.Fatalf("revoked acting session escaped transaction check: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("revoke-all did not complete after session revocation")
	}
	other, err := store.GetSession(ctx, otherID)
	if err != nil || other.IsBlocked {
		t.Fatalf("revoke-all invalidated other session with no authority: %+v err=%v", other, err)
	}
}

func TestConcurrentRevokeAllSerializesWithoutDeadlock(t *testing.T) {
	ctx, _, store := g12cStore(t)
	accountID, first, second := g12cSessions(t, ctx, store)
	done := make(chan error, 2)
	go func() { done <- RevokeAllSessions(ctx, store, accountID, first) }()
	go func() { done <- RevokeAllSessions(ctx, store, accountID, second) }()
	var succeeded, rejected int
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, ErrSessionUnavailable):
				rejected++
			default:
				t.Fatalf("concurrent revoke-all failed unexpectedly: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent revoke-all deadlocked")
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("revoke-all must commit exactly once: success=%d rejected=%d", succeeded, rejected)
	}
	for _, id := range []uuid.UUID{first, second} {
		s, err := store.GetSession(ctx, id)
		if err != nil || !s.IsBlocked {
			t.Fatalf("revoke-all left live session %+v err=%v", s, err)
		}
	}
}


func TestRevokeAllAccountLockPermitsForeignKeyKeyShare(t *testing.T) {
	ctx, pool, store := g12cStore(t)
	accountID, actingID, _ := g12cSessions(t, ctx, store)
	consent, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer consent.Release()
	writer, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Release()
	consentPID := int32(consent.Conn().PgConn().PID())
	writerPID := int32(writer.Conn().PgConn().PID())
	tx, err := consent.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var id uuid.UUID
	// OAuth code issuance holds FOR SHARE on the acting session before
	// inserting grants with an accounts FK KEY SHARE requirement.
	if err := tx.QueryRow(ctx, "SELECT id FROM sessions WHERE id=$1 FOR SHARE", actingID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		result <- RevokeAllSessions(ctx, sessionRevokeOnConn{SessionStore: store, conn: writer}, accountID, actingID)
	}()
	// Confirm revoke-all acquired the account lock and is blocked on the
	// consent session row, then test FK's KEY SHARE without a lock cycle.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	waiting := false
	for !waiting {
		select {
		case err := <-result:
			t.Fatalf("revoke-all unexpectedly completed while consent held session FOR SHARE: %v", err)
		case <-tick.C:
			if err := pool.QueryRow(ctx,
				"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
				consentPID, writerPID,
			).Scan(&waiting); err != nil {
				t.Fatal(err)
			}
		case <-timeout.C:
			t.Fatal("revoke-all did not wait on OAuth session FOR SHARE")
		}
	}
	keyCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var account int64
	if err := tx.QueryRow(keyCtx,
		"SELECT id FROM accounts WHERE id=$1 FOR KEY SHARE", accountID,
	).Scan(&account); err != nil || account != accountID {
		t.Fatalf("account lock blocked OAuth/foreign-key KEY SHARE: id=%d err=%v", account, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("revoke-all did not commit after consent released row: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("revoke-all deadlocked with OAuth session authority")
	}
}
