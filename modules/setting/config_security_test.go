package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExampleSigningKeyRejectedOutsideDevelopment(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.ini")
	data := "RUN_MODE = production\n[database]\nDB_SOURCE = postgresql://root:secret@localhost:5432/bdspro?sslmode=disable\nMIGRATION_URL = file://db/migration\n[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\nACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(file)
	if err == nil || !strings.Contains(err.Error(), "production signing key") {
		t.Fatalf("expected refusal, got %v", err)
	}
}
