package oauth

import (
	"context"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"time"
)

// Extends the existing oauthStub for the versioned consent contract.
func (s *oauthStub) GetOAuthConsentVersion(context.Context, int64, uuid.UUID) (int64, error) {
	return 0, nil
}
func (s *oauthStub) ApproveOAuthAuthCode(ctx context.Context, id uuid.UUID, account int64,
	redirect, scope, challenge string, digest []byte, expiry time.Time, version int64, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil { return db.ErrOAuthSessionInactive }
	if version != 0 {
		return db.ErrOAuthConsentStale
	}
	return s.CreateOAuthAuthCode(ctx, id, account, redirect, scope, challenge, digest, expiry)
}
