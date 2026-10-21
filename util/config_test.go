package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigPrefersDotEnv(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(
		"DB_SOURCE=postgres://fallback\nSERVER_ADDRESS=127.0.0.1:8080\n"+
			"TOKEN_SYMMETRIC_KEY="+strings.Repeat("a", 32)+"\n"+
			"ACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n",
	), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(
		"DB_SOURCE=postgres://local\nSERVER_ADDRESS=127.0.0.1:9090\n"+
			"TOKEN_SYMMETRIC_KEY="+strings.Repeat("b", 32)+"\n"+
			"ACCESS_TOKEN_DURATION=10m\nREFRESH_TOKEN_DURATION=48h\n",
	), 0600))
	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "postgres://local", config.DBSource)
	require.Equal(t, "127.0.0.1:9090", config.ServerAddress)
	require.Equal(t, 10*time.Minute, config.AccessTokenDuration)
}

func TestLoadConfigAppEnvFallback(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(
		"DB_SOURCE=postgres://fallback\nSERVER_ADDRESS=127.0.0.1:8080\n"+
			"TOKEN_SYMMETRIC_KEY="+strings.Repeat("a", 32)+"\n"+
			"ACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n",
	), 0600))
	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "postgres://fallback", config.DBSource)
}

func TestLoadConfigProcessEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(
		"DB_SOURCE=postgres://fallback\nSERVER_ADDRESS=127.0.0.1:8080\n"+
			"TOKEN_SYMMETRIC_KEY="+strings.Repeat("a", 32)+"\n"+
			"ACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n",
	), 0600))
	t.Setenv("DB_SOURCE", "postgres://environment")
	config, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "postgres://environment", config.DBSource)
}

func TestLoadConfigMissingValuesFails(t *testing.T) {
	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
}
