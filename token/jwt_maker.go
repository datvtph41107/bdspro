package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtIssuer   = "bdspro-auth"
	jwtAudience = "bdspro-api"
	jwtTTL      = 15 * time.Minute
)

var ErrInvalidToken = errors.New("invalid JWT")

// JWTMaker signs short-lived JWTs, each bound to an actual PostgreSQL
// opaque session. The signature alone does NOT grant tenant authorization.
type JWTMaker struct {
	key []byte
}

func NewJWTMaker(secret string) (*JWTMaker, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("TOKEN_SYMMETRIC_KEY must contain at least 32 bytes")
	}
	return &JWTMaker{key: []byte(secret)}, nil
}

func (s *JWTMaker) Issue(accountID int64, sessionToken string) (string, time.Time, error) {
	if accountID <= 0 || sessionToken == "" {
		return "", time.Time{}, ErrInvalidToken
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiry := now.Add(jwtTTL)
	sum := sha256.Sum256([]byte(sessionToken))
	claims := Payload{
		TokenType:   "access",
		SessionHash: hex.EncodeToString(sum[:]),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
			Subject:   strconv.FormatInt(accountID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
			ID:        hex.EncodeToString(nonce),
		},
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.key)
	if err != nil {
		return "", time.Time{}, err
	}
	return t, expiry, nil
}

func (s *JWTMaker) Verify(raw string) (int64, string, error) {
	if len(raw) > 4096 || raw == "" {
		return 0, "", ErrInvalidToken
	}
	var claims Payload
	t, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return s.key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer), jwt.WithAudience(jwtAudience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithNotBeforeRequired())
	if err != nil || t == nil || !t.Valid ||
		claims.TokenType != "access" || len(claims.SessionHash) != 64 ||
		claims.ID == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil ||
		claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) > jwtTTL+time.Second {
		return 0, "", ErrInvalidToken
	}
	accountID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || accountID <= 0 {
		return 0, "", ErrInvalidToken
	}
	if _, err := hex.DecodeString(claims.SessionHash); err != nil {
		return 0, "", ErrInvalidToken
	}
	return accountID, claims.SessionHash, nil
}
