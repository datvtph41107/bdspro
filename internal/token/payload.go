package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload contains the identity, purpose and lifetime of a JWT.
type Payload struct {
	ID        uuid.UUID `json:"id"`
	AccountID int64     `json:"account_id"`
	TokenType TokenType `json:"token_type"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewPayload creates a valid, uniquely identified access or refresh payload.
func NewPayload(accountID int64, duration time.Duration, kind TokenType) (*Payload, error) {
	if accountID <= 0 || duration <= 0 || (kind != TokenTypeAccess && kind != TokenTypeRefresh) {
		return nil, ErrInvalidToken
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Payload{
		ID:        id,
		AccountID: accountID,
		TokenType: kind,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}, nil
}

// Valid enforces the account identity, token purpose and expiration rules.
// jwt/v5 also validates the standard time claims during ParseWithClaims.
func (payload *Payload) Valid(kind TokenType) error {
	if payload == nil || payload.ID == uuid.Nil || payload.AccountID <= 0 ||
		(kind != TokenTypeAccess && kind != TokenTypeRefresh) || payload.TokenType != kind {
		return ErrInvalidToken
	}
	if !time.Now().Before(payload.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}

// The six getters implement jwt.Claims without embedding RegisteredClaims.
func (payload *Payload) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.ExpiredAt), nil
}

func (payload *Payload) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.IssuedAt), nil
}

func (payload *Payload) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.IssuedAt), nil
}

func (payload *Payload) GetIssuer() (string, error) {
	return "", nil
}

func (payload *Payload) GetSubject() (string, error) {
	return "", nil
}

func (payload *Payload) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{}, nil
}

var _ jwt.Claims = (*Payload)(nil)
