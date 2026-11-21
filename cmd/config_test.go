package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCheckCommand(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	content := "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = :8080\n" +
		"[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\n" +
		"ACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n" +
		"[database]\nDB_SOURCE = postgres://localhost/db\n[redis]\nADDR = localhost:6379\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	app := NewMainApp(AppVersion{Version: "test"})
	var output bytes.Buffer
	app.Writer = &output
	if err := app.Run(context.Background(), []string{"bdspro", "--config", file, "config", "check"}); err != nil {
		t.Fatalf("config check failed: %v", err)
	}
	if !strings.Contains(output.String(), "Configuration valid") {
		t.Fatalf("missing success output: %q", output.String())
	}
}
