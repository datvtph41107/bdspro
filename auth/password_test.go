package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgon2PasswordHash(t *testing.T) {
	hash, err := hashPassword("a-strong-test-password")
	require.NoError(t, err)
	require.True(t, checkPassword("a-strong-test-password", hash))
	require.False(t, checkPassword("a-different-password", hash))
	require.False(t, checkPassword("a-strong-test-password", "malformed"))
	require.False(t, checkPassword("a-strong-test-password", ""))

	another, err := hashPassword("a-strong-test-password")
	require.NoError(t, err)
	require.NotEqual(t, hash, another, "independent salts must produce different hashes")
	require.True(t, checkPassword("a-strong-test-password", another))
}

func TestTokenIsStoredAsDigest(t *testing.T) {
	token, hash, err := randomToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Len(t, hash, 64)
	require.NotEqual(t, token, hash)
	require.Equal(t, hash, digestToken(token))
}
