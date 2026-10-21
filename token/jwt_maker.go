package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeySize = 32

// JWTMaker creates and verifies JSON Web Tokens with HMAC SHA-256.
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker validates the signing key and implements Maker.
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d bytes", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

// CreateToken signs a new access or refresh token for an account.
func (maker *JWTMaker) CreateToken(
	accountID int64,
	duration time.Duration,
	kind TokenType,
) (string, *Payload, error) {
	payload, err := NewPayload(accountID, duration, kind)
	if err != nil {
		return "", nil, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}
	return token, payload, nil
}

// VerifyToken validates the HS256 signature, expiry and access/refresh type.
func (maker *JWTMaker) VerifyToken(raw string, kind TokenType) (*Payload, error) {
	if raw == "" || len(raw) > 4096 {
		return nil, ErrInvalidToken
	}

	keyFunc := func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(
		raw,
		&Payload{},
		keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok || !jwtToken.Valid {
		return nil, ErrInvalidToken
	}
	if err := payload.Valid(kind); err != nil {
		return nil, err
	}
	return payload, nil
}
