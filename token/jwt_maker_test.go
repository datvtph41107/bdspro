package token

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("a", 32))
	require.NoError(t, err)
	raw, payload, err := maker.CreateToken(17, time.Hour, TokenTypeAccess)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.NotEmpty(t, payload.ID)
	verified, err := maker.VerifyToken(raw, TokenTypeAccess)
	require.NoError(t, err)
	require.Equal(t, int64(17), verified.AccountID)
	_, err = maker.VerifyToken(raw, TokenTypeRefresh)
	require.Error(t, err)

	wrongKey, err := NewJWTMaker(strings.Repeat("b", 32))
	require.NoError(t, err)
	_, err = wrongKey.VerifyToken(raw, TokenTypeAccess)
	require.Error(t, err)

	expired, _, err := maker.CreateToken(17, time.Nanosecond, TokenTypeRefresh)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	_, err = maker.VerifyToken(expired, TokenTypeRefresh)
	require.Error(t, err)
	_, err = NewJWTMaker("short")
	require.Error(t, err)
}
