package auth

import "github.com/google/uuid"

// Actor is the verified API identity, not a user-provided claim.
// A session-backed access token is required before an Actor can be created.
type Actor struct {
	AccountID int64
	SessionID uuid.UUID
}
