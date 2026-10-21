package token

import (
    "errors"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Payload struct {
    ID        uuid.UUID `json:"id"`
    AccountID int64     `json:"account_id"`
    TokenType TokenType `json:"token_type"`
    jwt.RegisteredClaims
}

func NewPayload(accountID int64, duration time.Duration, kind TokenType) (*Payload, error) {
    if accountID <= 0 || duration <= 0 || (kind != TokenTypeAccess && kind != TokenTypeRefresh) {
        return nil, ErrInvalidToken
    }
    id, err := uuid.NewRandom()
    if err != nil { return nil, err }
    now := time.Now().UTC()
    return &Payload{
        ID: id, AccountID: accountID, TokenType: kind,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:  "account",
            IssuedAt: jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
        },
    }, nil
}
