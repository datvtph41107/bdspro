package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPositiveIDRouteParameter(t *testing.T) {
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{"1", true}, {"9223372036854775807", true},
		{"0", false}, {"-1", false}, {"abc", false},
		{"1.1", false}, {"", false}, {"9223372036854775808", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			value, err := PositiveID(tc.input)
			if tc.valid {
				require.NoError(t, err)
				require.Positive(t, value)
			} else {
				require.ErrorIs(t, err, ErrInvalid)
			}
		})
	}
}
