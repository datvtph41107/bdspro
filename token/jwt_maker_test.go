package token

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("a", 32))
	require.NoError(t, err)

	issuedAt := time.Now()
	duration := time.Minute

	raw, payload, err := maker.CreateToken(17, duration, TokenTypeAccess)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.NotNil(t, payload)
	require.NotZero(t, payload.ID)

	verified, err := maker.VerifyToken(raw, TokenTypeAccess)
	require.NoError(t, err)
	require.Equal(t, int64(17), verified.AccountID)
	require.Equal(t, TokenTypeAccess, verified.TokenType)
	require.Equal(t, payload.ID, verified.ID)
	require.WithinDuration(t, issuedAt, verified.IssuedAt, time.Second)
	require.WithinDuration(t, issuedAt.Add(duration), verified.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	secretKey := strings.Repeat("a", 32)
	maker, err := NewJWTMaker(secretKey)
	require.NoError(t, err)

	payload, err := NewPayload(17, time.Minute, TokenTypeAccess)
	require.NoError(t, err)
	payload.ExpiredAt = time.Now().Add(-time.Minute)

	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(secretKey))
	require.NoError(t, err)
	verified, err := maker.VerifyToken(raw, TokenTypeAccess)
	require.ErrorIs(t, err, ErrExpiredToken)
	require.Nil(t, verified)
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	payload, err := NewPayload(17, time.Minute, TokenTypeAccess)
	require.NoError(t, err)
	raw, err := jwt.NewWithClaims(jwt.SigningMethodNone, payload).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(strings.Repeat("a", 32))
	require.NoError(t, err)
	verified, err := maker.VerifyToken(raw, TokenTypeAccess)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, verified)
}

func TestJWTWrongTokenType(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("a", 32))
	require.NoError(t, err)
	raw, _, err := maker.CreateToken(17, time.Minute, TokenTypeAccess)
	require.NoError(t, err)
	verified, err := maker.VerifyToken(raw, TokenTypeRefresh)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, verified)
}

func TestJWTWrongSecretKey(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("a", 32))
	require.NoError(t, err)
	raw, _, err := maker.CreateToken(17, time.Minute, TokenTypeAccess)
	require.NoError(t, err)
	otherMaker, err := NewJWTMaker(strings.Repeat("b", 32))
	require.NoError(t, err)
	verified, err := otherMaker.VerifyToken(raw, TokenTypeAccess)
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, verified)
}

func TestJWTInvalidSecretKey(t *testing.T) {
	maker, err := NewJWTMaker("short")
	require.Error(t, err)
	require.Nil(t, maker)
}

func TestNewPayloadRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		accountID int64
		duration  time.Duration
		kind      TokenType
	}{
		{0, time.Minute, TokenTypeAccess},
		{17, 0, TokenTypeAccess},
		{17, -time.Minute, TokenTypeRefresh},
		{17, time.Minute, "unknown"},
	} {
		payload, err := NewPayload(tc.accountID, tc.duration, tc.kind)
		require.ErrorIs(t, err, ErrInvalidToken)
		require.Nil(t, payload)
	}
}
