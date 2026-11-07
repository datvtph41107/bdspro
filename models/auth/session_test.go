package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	affected   int64
	err        error
	auditErr   error
	auditCalls int
	seen       db.RevokeOwnedSessionParams
}

type testSessionTX struct {
	store *testSessionStore
}

func (tx testSessionTX) Exec(_ context.Context, statement string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(statement, "UPDATE sessions SET is_blocked") {
		tx.store.seen = db.RevokeOwnedSessionParams{ID: args[0].(uuid.UUID), AccountID: args[1].(int64)}
		if tx.store.err != nil {
			return pgconn.CommandTag{}, tx.store.err
		}
		return pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", tx.store.affected)), nil
	}
	if strings.Contains(statement, "INSERT INTO audit_events") {
		tx.store.auditCalls++
		if tx.store.auditErr != nil {
			return pgconn.CommandTag{}, tx.store.auditErr
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	return pgconn.CommandTag{}, fmt.Errorf("unexpected transaction SQL")
}

func (testSessionTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, fmt.Errorf("unexpected query")
}

func (testSessionTX) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func (s *testSessionStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	return fn(db.New(testSessionTX{store: s}))
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
	if store.auditCalls != 1 {
		t.Fatalf("successful revoke must write exactly one event: %d", store.auditCalls)
	}
	store.auditErr = errors.New("failed to audit")
	if err := RevokeOwnedSession(context.Background(), store, 42, id); !errors.Is(err, store.auditErr) {
		t.Fatalf("audit failure must fail owning transaction: %v", err)
	}
	if store.auditCalls != 2 {
		t.Fatalf("audit failure call was lost: %d", store.auditCalls)
	}
}

func TestRevokeAllSessionsPreservesStorageError(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	store := &testSessionStore{err: databaseErr}
	if err := RevokeAllSessions(context.Background(), store, 42); !errors.Is(err, databaseErr) {
		t.Fatalf("database error lost: %v", err)
	}
}

// OAuth authorization requires an active session authority even when its JWT
// signature and expiry are otherwise valid. PostgreSQL outages are not 401s.
type activeSessionReader struct {
	session db.Session
	err error
}
func (s activeSessionReader) GetSession(_ context.Context, _ uuid.UUID) (db.Session, error) {
	return s.session, s.err
}
func TestRequireActiveSession(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	id := uuid.New()
	live := db.Session{ID:id,AccountID:41,ExpiresAt:now.Add(time.Hour)}
	if err:=RequireActiveSession(ctx,activeSessionReader{session:live},41,id,now);err!=nil {
		t.Fatalf("live session rejected: %v",err)
	}
	for name, tc := range map[string]struct{
		session db.Session
		id uuid.UUID
		accountID int64
		err error
	}{
		"unbound JWT": {session:live,id:uuid.Nil,accountID:41},
		"wrong account": {session:live,id:id,accountID:42},
		"wrong session": {session:live,id:uuid.New(),accountID:41},
		"blocked": {session:db.Session{ID:id,AccountID:41,IsBlocked:true,ExpiresAt:now.Add(time.Hour)},id:id,accountID:41},
		"expired": {session:db.Session{ID:id,AccountID:41,ExpiresAt:now.Add(-time.Second)},id:id,accountID:41},
		"missing": {session:live,id:id,accountID:41,err:pgx.ErrNoRows},
	} {
		t.Run(name,func(t *testing.T){
			if err:=RequireActiveSession(ctx,activeSessionReader{session:tc.session,err:tc.err},tc.accountID,tc.id,now);!errors.Is(err,ErrSessionUnavailable) {
				t.Fatalf("inactive session accepted: %v",err)
			}
		})
	}
	unavailable:=errors.New("postgres unavailable")
	if err:=RequireActiveSession(ctx,activeSessionReader{err:unavailable},41,id,now);!errors.Is(err,unavailable) {
		t.Fatalf("DB outage was disguised as missing session: %v",err)
	}
}
