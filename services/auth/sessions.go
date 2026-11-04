package auth

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func (s *AuthService) ListSessions(ctx context.Context, accountID int64) ([]db.ListActiveSessionsRow, error) {
	return auth_model.ListSessions(ctx, s.store, accountID)
}

func (s *AuthService) RevokeOwnSession(ctx context.Context, accountID int64, sessionID uuid.UUID) error {
	return auth_model.RevokeOwnedSession(ctx, s.store, accountID, sessionID)
}

// RevokeAllSessions revokes refresh sessions only; access JWTs expire naturally.
func (s *AuthService) RevokeAllSessions(ctx context.Context, accountID int64) error {
	return auth_model.RevokeAllSessions(ctx, s.store, accountID)
}
