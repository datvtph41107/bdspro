package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExampleSigningKeyRejectedOutsideDevelopment(t *testing.T) {
	dir := t.TempDir()
	data := "ENVIRONMENT=production\nDB_SOURCE=postgresql://root:secret@localhost:5432/bdspro?sslmode=disable\nMIGRATION_URL=file://db/migration\nHTTP_SERVER_ADDRESS=127.0.0.1:8080\nTOKEN_SYMMETRIC_KEY=01234567890123456789012345678901\nACCESS_TOKEN_DURATION=15m\nREFRESH_TOKEN_DURATION=24h\n"
	if err := os.WriteFile(filepath.Join(dir, "app.env"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(dir)
	if err == nil || !strings.Contains(err.Error(), "production signing key") {
		t.Fatalf("expected sample secret refusal; got %v", err)
	}
}
