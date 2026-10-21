package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	password := "very-strong-test-password"

	hashedPassword1, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword1)

	require.NoError(t, CheckPassword(password, hashedPassword1))
	require.ErrorIs(t, CheckPassword("wrong-password", hashedPassword1), bcrypt.ErrMismatchedHashAndPassword)

	hashedPassword2, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword2)
	require.NotEqual(t, hashedPassword1, hashedPassword2)
}
