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

func TestLoadConfigValidation(t *testing.T) {
	cases := []struct {
		name, config, errorContains string
	}{
		{"driver", "DB_DRIVER=mysql", "DB_DRIVER must be postgres"},
		{"database", "DB_SOURCE=", "DB_SOURCE required"},
		{"address", "SERVER_ADDRESS=not-an-address", "invalid SERVER_ADDRESS"},
		{"unsafe_development_bind", "SERVER_ADDRESS=0.0.0.0:8080", "loopback listener"},
		{"production_smtp", "ENVIRONMENT=production", "SMTP_ADDR required"},
		{"production_origin", "ENVIRONMENT=production\nSMTP_ADDR=smtp.example.test:587", "ALLOWED_ORIGIN required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			contents := "ENVIRONMENT=development\nDB_DRIVER=postgres\nDB_SOURCE=postgresql://example.test/bdspro\nSERVER_ADDRESS=127.0.0.1:8080\n"
			contents += tc.config + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "app.env"), []byte(contents), 0600))
			_, err := LoadConfig(dir)
			require.ErrorContains(t, err, tc.errorContains)
		})
	}
}

func TestLoadProductionConfig(t *testing.T) {
	path := t.TempDir()
	contents := "ENVIRONMENT=production\nDB_DRIVER=postgres\nDB_SOURCE=postgresql://host.example.test/bdspro\nSERVER_ADDRESS=0.0.0.0:8080\nSMTP_ADDR=smtp.example.test:587\nALLOWED_ORIGIN=https://app.example.test\n"
	require.NoError(t, os.WriteFile(filepath.Join(path, "app.env"), []byte(contents), 0600))

	config, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "production", config.Environment)
	require.Equal(t, "smtp.example.test:587", config.SMTPAddr)
}
