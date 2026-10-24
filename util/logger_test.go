package util

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggerConsole(t *testing.T) {
	var output bytes.Buffer

	logger := NewLogger(&output, "console")
	logger.Info().
		Str("environment", "development").
		Msg("startup.begin")

	require.Contains(t, output.String(), "[INF]")
	require.Contains(t, output.String(), "\x1b[")
	require.Contains(t, output.String(), "startup.begin")
	require.Contains(t, output.String(), "development")
}

func TestNewLoggerJSON(t *testing.T) {
	var output bytes.Buffer

	logger := NewLogger(&output, "json")
	logger.Info().Msg("startup.begin")

	var result map[string]interface{}
	err := json.Unmarshal(output.Bytes(), &result)

	require.NoError(t, err)
	require.Equal(t, "info", result["level"])
	require.Equal(t, "startup.begin", result["message"])
}
