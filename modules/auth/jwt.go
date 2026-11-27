package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d bytes", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

func (maker *JWTMaker) CreateToken(
	accountID int64,
	duration time.Duration,
	kind TokenType,
) (string, *Payload, error) {
	payload, err := NewPayload(accountID, duration, kind)
	if err != nil {
		return "", nil, err
	}

	return maker.sign(payload)
}

func (maker *JWTMaker) CreateSessionAccessToken(accountID int64, duration time.Duration, sessionID uuid.UUID, sessionExpiry time.Time) (string, *Payload, error) {
	if sessionID == uuid.Nil {
		return "", nil, ErrInvalidToken
	}
	payload, err := NewPayload(accountID, duration, TokenTypeAccess)
	if err != nil {
		return "", nil, err
	}
	// Use the issuance timestamp already captured by NewPayload. Never extend
	// an access token beyond the persisted session's absolute deadline.
	if !sessionExpiry.After(payload.IssuedAt) {
		return "", nil, ErrExpiredToken
	}
	if sessionExpiry.Before(payload.ExpiredAt) {
		payload.ExpiredAt = sessionExpiry.UTC()
	}
	payload.SessionID = sessionID
	return maker.sign(payload)
}

func (maker *JWTMaker) sign(payload *Payload) (string, *Payload, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}
	return token, payload, nil
}

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
