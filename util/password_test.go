package util

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestPassword(t *testing.T) {
    password := "very-strong-test-password"
    hash1, err := HashPassword(password)
    require.NoError(t, err)
    require.NoError(t, CheckPassword(password, hash1))
    require.Error(t, CheckPassword("incorrect", hash1))
    hash2, err := HashPassword(password)
    require.NoError(t, err)
    require.NotEqual(t, hash1, hash2)
}
