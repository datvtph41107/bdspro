package util

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestRandomSecret(t *testing.T) {
	first, err := RandomSecret()
	require.NoError(t, err)
	second, err := RandomSecret()
	require.NoError(t, err)
	require.Len(t, first, 64)
	require.NotEqual(t, first, second)
}
