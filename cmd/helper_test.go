package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/urfave/cli/v3"
)

// Exercise the web configuration path without opening PostgreSQL or Redis:
// file logging must be ready before the first config event is emitted.
func TestWebInitializesFileLoggerBeforeConfigAnnouncement(t *testing.T) {
	workPath := t.TempDir()
	configFile := filepath.Join(workPath, "custom", "conf", "app.ini")
	if err := os.MkdirAll(filepath.Dir(configFile), 0700); err != nil {
		t.Fatal(err)
	}
	data := "RUN_MODE = development\n" +
		"[database]\nDB_SOURCE = postgresql://invalid:invalid@127.0.0.1:1/bdspro\n" +
		"MIGRATION_URL = file://db/migration\n" +
		"[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n" +
		"[redis]\nADDR = 127.0.0.1:6379\n" +
		"[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\n" +
		"ACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n" +
		"[log]\nMODE = file\nFORMAT = json\nLEVEL = info\nROOT_PATH = data/log\n" +
		"[log.file]\nFILE_NAME = web.log\n"
	if err := os.WriteFile(configFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := log.SetupStderrLogger(log.INFO); err != nil {
			t.Errorf("restore console logger: %v", err)
		}
	})
	app := NewMainApp(AppVersion{Version: "test"})
	var executed bool
	for _, c := range app.Commands {
		if c.Name == "web" {
			c.Action = func(_ context.Context, command *cli.Command) error {
				executed = true
				_, err := loadSettings(command)
				return err
			}
		}
	}
	if err := app.Run(context.Background(), []string{"bdspro", "--work-path", workPath, "web"}); err != nil {
		t.Fatalf("web setting bootstrap failed without DB access: %v", err)
	}
	if !executed {
		t.Fatal("web config action never executed")
	}
	logFile := filepath.Join(workPath, "data", "log", "web.log")
	content, err := os.ReadFile(logFile)
	if err != nil || !strings.Contains(string(content), "Using configuration file: ") {
		t.Fatalf("web config event was not emitted after configuring file logger: path=%s data=%q err=%v", logFile, content, err)
	}
}
