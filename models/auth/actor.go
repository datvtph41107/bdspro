package auth

import "github.com/google/uuid"

type Actor struct {
	AccountID int64
	SessionID uuid.UUID
}
