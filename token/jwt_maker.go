package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTMaker struct{ key []byte }

func NewJWTMaker(secret string) (Maker, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("token secret must contain at least 32 bytes")
	}
	return &JWTMaker{key: []byte(secret)}, nil
}

func (m *JWTMaker) CreateToken(accountID int64, duration time.Duration, kind TokenType) (string, *Payload, error) {
	payload, err := NewPayload(accountID, duration, kind)
	if err != nil {
		return "", nil, err
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString(m.key)
	return signed, payload, err
}

func (m *JWTMaker) VerifyToken(raw string, kind TokenType) (*Payload, error) {
	if raw == "" || len(raw) > 4096 {
		return nil, ErrInvalidToken
	}
	payload := &Payload{}
	parsed, err := jwt.ParseWithClaims(raw, payload,
		func(_ *jwt.Token) (any, error) { return m.key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || parsed == nil || !parsed.Valid ||
		payload.AccountID <= 0 || payload.ID == uuid.Nil || payload.TokenType != kind {
		return nil, ErrInvalidToken
	}
	return payload, nil
}
