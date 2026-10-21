package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeAppConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(content), 0600))
	return dir
}

func exampleAppConfig() string {
	return "DB_SOURCE=postgres://root:secret@localhost:5432/bdspro?sslmode=disable\n" +
		"SERVER_ADDRESS=127.0.0.1:8080\n" +
		"ALLOWED_ORIGINS=http://localhost:3000,https://example.com\n" +
		"TOKEN_SYMMETRIC_KEY=" + strings.Repeat("a", 32) + "\n" +
		"ACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n"
}

func TestLoadConfig(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	config, err := LoadConfig(writeAppConfig(t, exampleAppConfig()))
	require.NoError(t, err)
	require.Contains(t, config.DBSource, "/bdspro?")
	require.Equal(t, "127.0.0.1:8080", config.ServerAddress)
	require.Equal(t, 15*time.Minute, config.AccessTokenDuration)
	require.Equal(t, 24*time.Hour, config.RefreshTokenDuration)
	require.Equal(t, []string{"http://localhost:3000", "https://example.com"}, config.AllowedOrigins)
}

func TestLoadConfigProcessEnvOverridesFile(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	config, err := LoadConfig(writeAppConfig(t, exampleAppConfig()))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9090", config.ServerAddress)
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
}

func TestLoadConfigInvalidKey(t *testing.T) {
	for _, key := range []string{"DB_SOURCE", "SERVER_ADDRESS", "TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION"} {
		t.Setenv(key, "")
	}
	configFile := strings.Replace(exampleAppConfig(), strings.Repeat("a", 32), "short", 1)
	_, err := LoadConfig(writeAppConfig(t, configFile))
	require.Error(t, err)
}
