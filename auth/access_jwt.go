package auth

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
	accessIssuer   = "bdspro-auth"
	accessAudience = "bdspro-api"
	accessTTL      = 15 * time.Minute
)

var ErrAccessToken = errors.New("invalid access token")

// AccessTokens signs short-lived JWTs, each bound to an actual PostgreSQL
// opaque session. The signature alone does NOT grant tenant authorization.
type AccessTokens struct {
	key []byte
}

type accessClaims struct {
	TokenType   string `json:"token_type"`
	SessionHash string `json:"sid"`
	jwt.RegisteredClaims
}

func NewAccessTokens(secret string) (*AccessTokens, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("ACCESS_TOKEN_KEY must contain at least 32 bytes")
	}
	return &AccessTokens{key: []byte(secret)}, nil
}

func (s *AccessTokens) Issue(accountID int64, sessionToken string) (string, time.Time, error) {
	if accountID <= 0 || sessionToken == "" {
		return "", time.Time{}, ErrAccessToken
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiry := now.Add(accessTTL)
	sum := sha256.Sum256([]byte(sessionToken))
	claims := accessClaims{
		TokenType:   "access",
		SessionHash: hex.EncodeToString(sum[:]),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessIssuer,
			Audience:  jwt.ClaimStrings{accessAudience},
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

func (s *AccessTokens) Verify(raw string) (int64, string, error) {
	if len(raw) > 4096 || raw == "" {
		return 0, "", ErrAccessToken
	}
	var claims accessClaims
	t, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return s.key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(accessIssuer), jwt.WithAudience(accessAudience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithNotBeforeRequired())
	if err != nil || t == nil || !t.Valid ||
		claims.TokenType != "access" || len(claims.SessionHash) != 64 ||
		claims.ID == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil ||
		claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) > accessTTL+time.Second {
		return 0, "", ErrAccessToken
	}
	accountID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || accountID <= 0 {
		return 0, "", ErrAccessToken
	}
	if _, err := hex.DecodeString(claims.SessionHash); err != nil {
		return 0, "", ErrAccessToken
	}
	return accountID, claims.SessionHash, nil
}
