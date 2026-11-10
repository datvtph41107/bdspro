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

func TestMigrationRelativeLogRootUsesApplicationWorkPath(t *testing.T) {
	workPath := t.TempDir()
	configFile := filepath.Join(workPath, "custom", "conf", "app.ini")
	if err := os.MkdirAll(filepath.Dir(configFile), 0700); err != nil {
		t.Fatal(err)
	}
	content := "RUN_MODE = development\n" +
		"[database]\nDB_SOURCE = postgresql://invalid:invalid@127.0.0.1:1/bdspro\n" +
		"MIGRATION_URL = file://db/migration\n" +
		"[log]\nMODE = file\nFORMAT = json\nLEVEL = info\nROOT_PATH = data/log\n" +
		"[log.file]\nFILE_NAME = migration.log\n"
	if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := log.SetupStderrLogger(log.INFO); err != nil {
			t.Errorf("restore console logger: %v", err)
		}
	})
	app := NewMainApp(AppVersion{Version: "test"})
	err := app.Run(context.Background(), []string{
		"bdspro", "--work-path", workPath, "migrate", "--steps=-1",
	})
	if err == nil || !strings.Contains(err.Error(), "--steps cannot be negative") {
		t.Fatalf("migration reached dependencies instead of validating arguments: %v", err)
	}
	logFile := filepath.Join(workPath, "data", "log", "migration.log")
	data, err := os.ReadFile(logFile)
	if err != nil || !strings.Contains(string(data), "Using configuration file: ") {
		t.Fatalf("relative log root did not resolve under work path: file=%q data=%q err=%v", logFile, data, err)
	}
}
