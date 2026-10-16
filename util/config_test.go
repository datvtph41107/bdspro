package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	path := t.TempDir()
	contents := "ENVIRONMENT=development\nDB_DRIVER=postgres\nDB_SOURCE=postgresql://file-host/bdspro\nSERVER_ADDRESS=127.0.0.1:8080\nALLOWED_ORIGIN=\n"
	require.NoError(t, os.WriteFile(filepath.Join(path, "app.env"), []byte(contents), 0600))

	config, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "development", config.Environment)
	require.Equal(t, "postgres", config.DBDriver)
	require.Equal(t, "postgresql://file-host/bdspro", config.DBSource)
	require.Equal(t, "127.0.0.1:8080", config.ServerAddress)

	t.Setenv("DB_SOURCE", "postgresql://override-host/bdspro")
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	t.Setenv("ALLOWED_ORIGIN", "https://example.test")
	override, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "postgresql://override-host/bdspro", override.DBSource)
	require.Equal(t, "127.0.0.1:9090", override.ServerAddress)
	require.Equal(t, "https://example.test", override.AllowedOrigin)
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(t.TempDir())
	require.Error(t, err)
}
