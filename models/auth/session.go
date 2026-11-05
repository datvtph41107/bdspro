package auth

import (
	"context"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/timeutil"
	"github.com/google/uuid"
)

// SessionStore is the persistence capability consumed by session operations.
type SessionStore interface {
	CreateSessionWithConnection(context.Context, db.CreateSessionParams) (db.Session, error)
	GetSession(context.Context, uuid.UUID) (db.Session, error)
	ListActiveSessions(context.Context, int64) ([]db.ListActiveSessionsRow, error)
	RevokeOwnedSession(context.Context, db.RevokeOwnedSessionParams) (int64, error)
	RevokeAllAccountSessions(context.Context, int64) (int64, error)
}

func NewSession(ctx context.Context, store SessionStore, params db.CreateSessionParams) (db.Session, error) {
	return store.CreateSessionWithConnection(ctx, params)
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
