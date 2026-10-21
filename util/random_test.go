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

func TestRandomHelpers(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := RandomInt(10, 20)
		require.GreaterOrEqual(t, n, int64(10))
		require.LessOrEqual(t, n, int64(20))
	}
	require.Len(t, RandomString(12), 12)
	require.Len(t, RandomPersonName(), 6)
	require.Len(t, RandomOrganizationName(), 10)
	require.Contains(t, RandomEmail(), "@example.test")
}
