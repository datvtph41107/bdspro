package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestValidRefreshSession(t *testing.T) {
	now := time.Now()
	session := db.Session{AccountID: 7, RefreshToken: "token", ExpiresAt: now.Add(time.Minute)}
	if !ValidRefreshSession(session, 7, "token", now) {
		t.Fatal("legacy valid session rejected")
	}
	session.RefreshToken = RefreshTokenDigest("token")
	if !ValidRefreshSession(session, 7, "token", now) || ValidRefreshSession(session, 7, "wrong", now) {
		t.Fatal("hashed refresh proof authority failed")
	}
	session.IsBlocked = true
	if ValidRefreshSession(session, 7, "token", now) {
		t.Fatal("blocked session accepted")
	}
	session.IsBlocked = false
	if ValidRefreshSession(session, 8, "token", now) ||
		ValidRefreshSession(session, 7, "other", now) ||
		ValidRefreshSession(session, 7, "token", now.Add(2*time.Minute)) {
		t.Fatal("session authority/expiry check failed")
	}
}

type testSessionStore struct {
	SessionStore
	affected int64
	err      error
	seen     db.RevokeOwnedSessionParams
}

func (s *testSessionStore) RevokeOwnedSession(_ context.Context, arg db.RevokeOwnedSessionParams) (int64, error) {
	s.seen = arg
	return s.affected, s.err
}

func (s *testSessionStore) RevokeAllAccountSessions(_ context.Context, _ int64) (int64, error) {
	return s.affected, s.err
}

func TestRevokeOwnedSession(t *testing.T) {
	id := uuid.New()
	store := &testSessionStore{}
	err := RevokeOwnedSession(context.Background(), store, 42, id)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unknown account session accepted: %v", err)
	}
	if store.seen.AccountID != 42 || store.seen.ID != id {
		t.Fatalf("incorrect ownership predicate: %+v", store.seen)
	}

	store.affected = 1
	if err := RevokeOwnedSession(context.Background(), store, 42, id); err != nil {
		t.Fatalf("owned session cannot be revoked: %v", err)
	}
}

func TestRevokeAllSessionsPreservesStorageError(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	store := &testSessionStore{err: databaseErr}
	if err := RevokeAllSessions(context.Background(), store, 42); !errors.Is(err, databaseErr) {
		t.Fatalf("database error lost: %v", err)
	}
}
