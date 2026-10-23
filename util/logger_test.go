package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggerDevelopmentColorsLevelOnly(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	var output bytes.Buffer
	logger := NewLogger(&output, "development")
	logger.Info().Str("environment", "development").Msg("startup.begin")
	line := output.String()
	require.Contains(t, line, "\x1b[32mINF\x1b[0m")
	require.Contains(t, line, "startup.begin")
	require.Contains(t, line, "environment=development")
	require.Equal(t, 2, strings.Count(line, "\x1b["))
}

func TestNewLoggerNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var output bytes.Buffer
	logger := NewLogger(&output, "development")
	logger.Warn().Msg("warning")
	require.Contains(t, output.String(), "WRN")
	require.NotContains(t, output.String(), "\x1b[")
}

func TestNewLoggerProductionIsJSON(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(&output, "production")
	logger.Info().Msg("startup.begin")
	require.Contains(t, output.String(), "\"level\":\"info\"")
	require.NotContains(t, output.String(), "\x1b[")
}
