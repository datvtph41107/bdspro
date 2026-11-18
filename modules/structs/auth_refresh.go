package structs

import "github.com/google/uuid"

// RefreshSessionOption identifies the session and proves possession of its
// single-use opaque refresh credential. The session UUID alone is insufficient.
type RefreshSessionOption struct {
	SessionID    uuid.UUID `json:"session_id" binding:"required"`
	RefreshToken string    `json:"refresh_token" binding:"required,min=43,max=43"`
}
