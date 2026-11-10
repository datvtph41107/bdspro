package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/log"
)

// A rejected migration flag is a DB-free probe: it occurs after logging is
// initialized, but before any migration or PostgreSQL connection is opened.
func TestMigrateInitializesConfiguredLoggerBeforeDatabaseAction(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "app.ini")
	content := fmt.Sprintf("RUN_MODE = development\n[database]\nDB_SOURCE = postgresql://invalid:invalid@127.0.0.1:1/bdspro\nMIGRATION_URL = file://db/migration\n[log]\nMODE = file\nFORMAT = json\nLEVEL = info\nROOT_PATH = %s\n[log.file]\nFILE_NAME = migrate.log\n", dir)
	if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := log.SetupStderrLogger(log.INFO); err != nil {
			t.Errorf("restore console logger: %v", err)
		}
	})
	app := NewMainApp(AppVersion{Version: "test"})
	err := app.Run(context.Background(), []string{"bdspro", "--config", configFile, "migrate", "--steps=-1"})
	if err == nil || !strings.Contains(err.Error(), "--steps cannot be negative") {
		t.Fatalf("expected flag rejection before connecting to PostgreSQL, got: %v", err)
	}
	contentOnDisk, err := os.ReadFile(filepath.Join(dir, "migrate.log"))
	if err != nil {
		t.Fatalf("migration did not initialize its configured log output: %v", err)
	}
	if !strings.Contains(string(contentOnDisk), "Using configuration file: ") {
		t.Fatalf("migration did not log through configured sink: %s", contentOnDisk)
	}
}
