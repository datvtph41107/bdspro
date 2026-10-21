package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRandomSecret(t *testing.T) {
	first, err := RandomSecret()
	require.NoError(t, err)
	second, err := RandomSecret()
	require.NoError(t, err)
	require.Len(t, first, 64)
	require.NotEqual(t, first, second)
}
