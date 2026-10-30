package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogLevelDefaultAndOverride(t *testing.T) {
	directory := t.TempDir()
	env := strings.Join([]string{
		"ENVIRONMENT=development",
		"DB_SOURCE=postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable",
		"MIGRATION_URL=file://db/migration",
		"HTTP_SERVER_ADDRESS=127.0.0.1:8080",
		"TOKEN_SYMMETRIC_KEY=01234567890123456789012345678901",
		"ACCESS_TOKEN_DURATION=15m",
		"REFRESH_TOKEN_DURATION=24h",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(directory, "app.env"), []byte(env), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("default LOG_LEVEL = %q, want info", cfg.LogLevel)
	}

	t.Setenv("LOG_LEVEL", "debug")
	cfg, err = LoadConfig(directory)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("environment LOG_LEVEL = %q, want debug", cfg.LogLevel)
	}

	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := LoadConfig(directory); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("expected invalid LOG_LEVEL rejection, got %v", err)
	}
}
