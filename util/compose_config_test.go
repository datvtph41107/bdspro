package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainerDevelopmentBindRequiresExplicitOptIn(t *testing.T) {
	c := Config{
		Environment: "development", DBDriver: "postgres",
		DBSource: "postgresql://example/bdspro", ServerAddress: "0.0.0.0:8080",
	}
	require.ErrorContains(t, c.validate(), "loopback listener")
	c.AllowContainerDevBind = true
	require.NoError(t, c.validate())

	c.Environment = "production"
	require.ErrorContains(t, c.validate(), "only allowed in development")
}
