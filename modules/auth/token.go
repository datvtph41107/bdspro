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
	// CreateSessionAccessToken binds newly issued access JWTs to the canonical
	// refresh session. Legacy JWTs remain readable by VerifyToken.
	CreateSessionAccessToken(accountID int64, duration time.Duration, sessionID uuid.UUID) (string, *Payload, error)
	VerifyToken(raw string, kind TokenType) (*Payload, error)
}
