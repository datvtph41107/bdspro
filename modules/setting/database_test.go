package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDatabaseSettingsDoesNotRequireHTTPOrJWT(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	data := "RUN_MODE = development\n[database]\nDB_SOURCE = postgresql://example:5432/bdspro\nMIGRATION_URL = file://db/migration\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadDatabaseSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBSource != "postgresql://example:5432/bdspro" {
		t.Fatalf("source: %q", cfg.DBSource)
	}
}

func TestMigrationSettingsUsesDatabaseCommandLoggerPolicy(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	data := "RUN_MODE = development\n[database]\nDB_SOURCE = postgresql://example:5432/bdspro\nMIGRATION_URL = file://db/migration\n[log]\nLEVEL = debug\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadDatabaseSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Level != "debug" || cfg.Log.Mode != "console" {
		t.Fatalf("migration logging policy is not shared: %+v", cfg.Log)
	}
	t.Setenv("BDS__LOG__LEVEL", "verbose")
	if _, err := LoadDatabaseSettings(file); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("migration accepted invalid logger configuration: %v", err)
	}
}
