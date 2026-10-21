package token

import "github.com/golang-jwt/jwt/v5"

// Payload is the signed JWT claim set. A valid signature does not by itself
// authorize any BDSPro operation: the PostgreSQL Session (sid) must still
// exist and remain active, and organization rights must be queried live.
// TokenType distinguishes short-lived access JWTs from other proof types.
type Payload struct {
	TokenType   string `json:"token_type"`
	SessionHash string `json:"sid"`
	jwt.RegisteredClaims
}
