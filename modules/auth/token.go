package auth

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Maker interface {
	CreateToken(accountID int64, duration time.Duration, kind TokenType) (string, *Payload, error)

	CreateSessionAccessToken(accountID int64, duration time.Duration, sessionID uuid.UUID, sessionExpiry time.Time) (string, *Payload, error)
	VerifyToken(raw string, kind TokenType) (*Payload, error)
}
