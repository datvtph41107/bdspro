package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMaintenanceConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDatabaseCommandSettingsDoNotRequireHTTPOrMigration(t *testing.T) {
	file := writeMaintenanceConfig(t, "[database]\nDB_SOURCE = postgres://user:password@localhost/bdspro\n")
	settings, err := LoadDatabaseCommandSettings(file)
	if err != nil {
		t.Fatal(err)
	}
	if settings.DBSource == "" || settings.Log.Level != "info" {
		t.Fatalf("invalid database command defaults: %+v", settings)
	}
	if _, err := LoadConfig(file); err == nil {
		t.Fatal("HTTP settings unexpectedly accepted without security config")
	}
}

func TestCleanupSettingsScopeAndRetention(t *testing.T) {
	base := "[database]\nDB_SOURCE = postgres://localhost/bdspro\n[audit]\nRETENTION_DAYS = "
	for _, tc := range []struct {
		name, days string
		valid      bool
	}{
		{"minimum", "1", true},
		{"maximum", "3650", true},
		{"zero", "0", false},
		{"too_large", "3651", false},
		{"malformed", "abc", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := writeMaintenanceConfig(t, base+tc.days+"\n")
			settings, err := LoadCleanupSettings(file)
			if (err == nil) != tc.valid {
				t.Fatalf("settings=%+v error=%v", settings, err)
			}
		})
	}
	file := writeMaintenanceConfig(t, "[database]\nDB_SOURCE = postgres://localhost/bdspro\n")
	settings, err := LoadCleanupSettings(file)
	if err != nil || settings.AuditRetentionDays != 90 {
		t.Fatalf("default retention=%+v error=%v", settings, err)
	}
}

func TestMaintenanceSettingsShareLogValidationAndOverrides(t *testing.T) {
	file := writeMaintenanceConfig(t, "[database]\nDB_SOURCE = postgres://localhost/bdspro\n")
	t.Setenv("BDS__LOG__LEVEL", "debug")
	settings, err := LoadDatabaseCommandSettings(file)
	if err != nil || settings.Log.Level != "debug" {
		t.Fatalf("log override=%+v err=%v", settings, err)
	}
	t.Setenv("BDS__LOG__LEVEL", "verbose")
	if _, err := LoadDatabaseCommandSettings(file); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("bad log level accepted: %v", err)
	}
}

func TestDatabaseMaintenanceDoesNotRequireAuditKey(t *testing.T) {
	file := writeMaintenanceConfig(t, "RUN_MODE = production\n[database]\nDB_SOURCE = postgres://localhost/bdspro\n")
	if _, err := LoadDatabaseCommandSettings(file); err != nil {
		t.Fatalf("doctor incorrectly requires production HTTP secrets: %v", err)
	}
	if _, err := LoadCleanupSettings(file); err != nil {
		t.Fatalf("cleanup incorrectly requires production HTTP secrets: %v", err)
	}
}
