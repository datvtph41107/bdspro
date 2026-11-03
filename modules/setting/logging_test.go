package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogLevelDefaultAndOverride(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	data := "RUN_MODE = development\n[database]\nDB_SOURCE = postgresql://root:secret@localhost:5432/bdspro?sslmode=disable\nMIGRATION_URL = file://db/migration\n[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\nACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("default log level=%q", cfg.LogLevel)
	}
	t.Setenv("BDS__LOG__LEVEL", "debug")
	cfg, err = LoadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("env override=%q", cfg.LogLevel)
	}
	t.Setenv("BDS__LOG__LEVEL", "verbose")
	if _, err = LoadConfig(file); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("invalid log level accepted: %v", err)
	}
}
