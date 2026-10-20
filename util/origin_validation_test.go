package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowedOriginMustBeExactOrigin(t *testing.T) {
	base := Config{
		Environment: "development", DBDriver: "postgres",
		DBSource: "postgresql://localhost/bdspro", ServerAddress: "127.0.0.1:8080",
	}
	for _, origin := range []string{
		"https://example.test/path", "https://example.test/",
		"https://example.test?x=1", "https://example.test/#fragment",
		"https://user:pass@example.test", "*", "null",
		"https://*.example.test", "https://example.test another",
		"javascript:alert(1)", "ftp://example.test",
	} {
		t.Run(origin, func(t *testing.T) {
			c := base
			c.AllowedOrigin = origin
			require.ErrorContains(t, c.validate(), "ALLOWED_ORIGIN")
		})
	}
	for _, origin := range []string{
		"http://localhost:3000", "http://127.0.0.1:18080",
		"https://app.example.test",
	} {
		t.Run("valid_"+origin, func(t *testing.T) {
			c := base
			c.AllowedOrigin = origin
			require.NoError(t, c.validate())
		})
	}
	c := base
	c.Environment = "production"
	c.AllowedOrigin = "http://app.example.test"
	require.ErrorContains(t, c.validate(), "HTTPS")
}
