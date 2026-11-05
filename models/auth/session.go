package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/timeutil"
	"github.com/google/uuid"
)

// SessionStore provides session reads, revocations and the shared SQLC transaction primitive.
// Authentication owns the session + connection-history commit invariant.
type SessionStore interface {
	db.Transactioner
	GetSession(context.Context, uuid.UUID) (db.Session, error)
	ListActiveSessions(context.Context, int64) ([]db.ListActiveSessionsRow, error)
	RevokeOwnedSession(context.Context, db.RevokeOwnedSessionParams) (int64, error)
	RevokeAllAccountSessions(context.Context, int64) (int64, error)
}

func NewSession(ctx context.Context, store SessionStore, params db.CreateSessionParams) (db.Session, error) {
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.Session, error) {
		session, err := q.CreateSession(ctx, params)
		if err != nil {
			return db.Session{}, err
		}
		if err := q.CreateAccountConnectionHistory(ctx, db.CreateAccountConnectionHistoryParams{
			AccountID: params.AccountID,
			SessionID: session.ID,
			IpAddress: params.ClientIp,
			UserAgent: params.UserAgent,
		}); err != nil {
			return db.Session{}, err
		}
		return session, nil
	})
}

func ValidRefreshSession(session db.Session, accountID int64, refreshToken string, now time.Time) bool {
	return !session.IsBlocked && session.AccountID == accountID &&
		MatchesRefreshToken(session.RefreshToken, refreshToken) && !timeutil.Expired(now, session.ExpiresAt)
}

func ListSessions(ctx context.Context, store SessionStore, accountID int64) ([]db.ListActiveSessionsRow, error) {
	return store.ListActiveSessions(ctx, accountID)
}

func RevokeOwnedSession(ctx context.Context, store SessionStore, accountID int64, id uuid.UUID) error {
	affected, err := store.RevokeOwnedSession(ctx, db.RevokeOwnedSessionParams{ID: id, AccountID: accountID})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrSessionNotFound
	}
	return nil
}

func RevokeAllSessions(ctx context.Context, store SessionStore, accountID int64) error {
	_, err := store.RevokeAllAccountSessions(ctx, accountID)
	return err
}

const refreshTokenDigestPrefix = "sha256:"

// RefreshTokenDigest stores a one-way fingerprint of a high-entropy signed JWT.
// Clients receive the original bearer token; database sessions persist only this
// fingerprint for newly issued credentials.
func RefreshTokenDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return refreshTokenDigestPrefix + hex.EncodeToString(digest[:])
}

// MatchesRefreshToken supports both fingerprinted sessions and legacy raw JWT
// rows during a bounded migration window. JWT validity and account/session
// authority are checked separately by the caller before this comparison.
func MatchesRefreshToken(stored, raw string) bool {
	if stored == "" || raw == "" {
		return false
	}
	expected := raw
	if strings.HasPrefix(stored, refreshTokenDigestPrefix) {
		expected = RefreshTokenDigest(raw)
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(expected)) == 1
}
