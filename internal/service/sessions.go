package service

import (
	"context"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/google/uuid"
)

func (s *AuthService) ListSessions(ctx context.Context, accountID int64) ([]db.ListActiveSessionsRow, error) {
	return s.store.ListActiveSessions(ctx, accountID)
}

func (s *AuthService) RevokeOwnSession(ctx context.Context, accountID int64, sessionID uuid.UUID) error {
	affected, err := s.store.RevokeOwnedSession(ctx, db.RevokeOwnedSessionParams{
		ID: sessionID, AccountID: accountID,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return apperrors.SessionNotFound
	}
	return nil
}

// RevokeAllSessions revokes refresh sessions only. Stateless access JWTs are
// not invalidated until their standard expiry; this limitation is explicit.
func (s *AuthService) RevokeAllSessions(ctx context.Context, accountID int64) error {
	_, err := s.store.RevokeAllAccountSessions(ctx, accountID)
	return err
}
