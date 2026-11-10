package auth

import (
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

// These tests use the same disposable PostgreSQL contract as the revoke-all
// tests. They intentionally pin connections and observe real lock waits.
func TestRevokeOwnedCannotOutliveActingSession(t *testing.T) {
	ctx, pool, store := g12cStore(t)
	accountID, actorID, targetID := g12cSessions(t, ctx, store)

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

	tx, err := revoker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "UPDATE sessions SET is_blocked=TRUE WHERE id=$1", actorID); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- RevokeOwnedSession(db.WithStore(ctx, sessionRevokeOnConn{SessionStore: store, conn: writer}), accountID, actorID, targetID)
	}()
	revokerPID := int32(revoker.Conn().PgConn().PID())
	writerPID := int32(writer.Conn().PgConn().PID())
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	blocked := false
	for !blocked {
		select {
		case err := <-done:
			t.Fatalf("revoke completed before actor authority resolved: %v", err)
		case <-ticker.C:
			if err := pool.QueryRow(ctx,
				"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
				revokerPID, writerPID).Scan(&blocked); err != nil {
				t.Fatal(err)
			}
		case <-timeout.C:
			t.Fatal("revoke did not wait for revoking actor")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrSessionUnavailable) {
			t.Fatalf("revoked actor escaped transaction: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("owned revoke did not finish after lock release")
	}
	target, err := store.GetSession(ctx, targetID)
	if err != nil || target.IsBlocked {
		t.Fatalf("invalid actor changed target: %+v err=%v", target, err)
	}
}

func TestCrossRevokeOwnedSessionsSerializeWithoutDeadlock(t *testing.T) {
	ctx, _, store := g12cStore(t)
	accountID, first, second := g12cSessions(t, ctx, store)
	done := make(chan error, 2)
	go func() { done <- RevokeOwnedSession(db.WithStore(ctx, store), accountID, first, second) }()
	go func() { done <- RevokeOwnedSession(db.WithStore(ctx, store), accountID, second, first) }()
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
				t.Fatalf("cross-revoke failed unexpectedly: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cross-revoke deadlocked")
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("cross-revoke must have exactly one winner: success=%d rejected=%d", succeeded, rejected)
	}
	var blocked int
	for _, id := range []uuid.UUID{first, second} {
		session, err := store.GetSession(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if session.IsBlocked {
			blocked++
		}
	}
	if blocked != 1 {
		t.Fatalf("cross-revoke changed %d sessions; expected one", blocked)
	}
}

func TestRevokeOwnedAndAllShareLockOrder(t *testing.T) {
	ctx, _, store := g12cStore(t)
	accountID, first, second := g12cSessions(t, ctx, store)
	done := make(chan error, 2)
	go func() { done <- RevokeOwnedSession(db.WithStore(ctx, store), accountID, first, second) }()
	go func() { done <- RevokeAllSessions(db.WithStore(ctx, store), accountID, second) }()
	var success, rejected int
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			switch {
			case err == nil:
				success++
			case errors.Is(err, ErrSessionUnavailable):
				rejected++
			default:
				t.Fatalf("owned/all revoke failed unexpectedly: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("owned/all revoke deadlocked")
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("owned/all must serialize authority: success=%d rejected=%d", success, rejected)
	}
}

func TestRevokeOwnedDoesNotRevealAnotherAccountSession(t *testing.T) {
	ctx, _, store := g12cStore(t)
	accountID, actingID, _ := g12cSessions(t, ctx, store)
	_, foreignID, _ := g12cSessions(t, ctx, store)
	if err := RevokeOwnedSession(db.WithStore(ctx, store), accountID, actingID, foreignID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("foreign session not masked as missing: %v", err)
	}
	foreign, err := store.GetSession(ctx, foreignID)
	if err != nil || foreign.IsBlocked {
		t.Fatalf("foreign session changed: %+v err=%v", foreign, err)
	}
	if err := RevokeOwnedSession(db.WithStore(ctx, store), accountID, actingID, actingID); err != nil {
		t.Fatalf("self-revocation failed: %v", err)
	}
	self, err := store.GetSession(ctx, actingID)
	if err != nil || !self.IsBlocked {
		t.Fatalf("self-revocation not committed: %+v err=%v", self, err)
	}
	// A second request from the same, now-invalid actor cannot revoke.
	if err := RevokeOwnedSession(db.WithStore(ctx, store), accountID, actingID, uuid.New()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("revoked actor retained authority: %v", err)
	}
}

var _ db.Transactioner = sessionRevokeOnConn{}
