package token

import "time"

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Maker interface {
	CreateToken(accountID int64, duration time.Duration, kind TokenType) (string, *Payload, error)
	VerifyToken(raw string, kind TokenType) (*Payload, error)
}
