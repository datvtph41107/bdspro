package util

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggerDevelopment(t *testing.T) {
	var output bytes.Buffer

	logger := NewLogger(&output, "development")
	logger.Info().
		Str("environment", "development").
		Msg("startup.begin")

	require.Contains(t, output.String(), "INF")
	require.Contains(t, output.String(), "startup.begin")
	require.Contains(t, output.String(), "development")
}

func TestNewLoggerProduction(t *testing.T) {
	var output bytes.Buffer

	logger := NewLogger(&output, "production")
	logger.Info().Msg("startup.begin")

	var result map[string]interface{}
	err := json.Unmarshal(output.Bytes(), &result)

	require.NoError(t, err)
	require.Equal(t, "info", result["level"])
	require.Equal(t, "startup.begin", result["message"])
}
