package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	minSecretKeySize = 32
	accessTokenType  = "access"
)

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d bytes", minSecretKeySize)
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

func (maker *JWTMaker) CreateSessionAccessToken(accountID int64, duration time.Duration, sessionID uuid.UUID, sessionExpiry time.Time) (string, *Payload, error) {
	if accountID <= 0 || duration <= 0 || sessionID == uuid.Nil {
		return "", nil, ErrInvalidToken
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	if !sessionExpiry.After(now) {
		return "", nil, ErrExpiredToken
	}
	expiry := now.Add(duration)
	if sessionExpiry.Before(expiry) {
		expiry = sessionExpiry.UTC()
	}
	payload := &Payload{
		ID:        id,
		SessionID: sessionID,
		AccountID: accountID,
		TokenType: accessTokenType,
		IssuedAt:  now,
		ExpiredAt: expiry,
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}
	return raw, payload, nil
}

func (maker *JWTMaker) VerifyAccessToken(raw string) (*Payload, error) {
	if raw == "" || len(raw) > 4096 {
		return nil, ErrInvalidToken
	}
	parsed, err := jwt.ParseWithClaims(
		raw,
		&Payload{},
		func(*jwt.Token) (any, error) { return []byte(maker.secretKey), nil },
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

	payload, ok := parsed.Claims.(*Payload)
	if !ok || !parsed.Valid || payload == nil || payload.ID == uuid.Nil ||
		payload.AccountID <= 0 || payload.SessionID == uuid.Nil || payload.TokenType != accessTokenType {
		return nil, ErrInvalidToken
	}
	// Keep the full timestamp check in addition to the JWT library's NumericDate validation.
	if !time.Now().Before(payload.ExpiredAt) {
		return nil, ErrExpiredToken
	}
	return payload, nil
}
