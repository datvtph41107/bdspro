package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedLoggerWritesFormattedMessage(t *testing.T) {
	m := NewManager()
	var out bytes.Buffer
	if err := m.Configure(&out, "json", INFO); err != nil {
		t.Fatal(err)
	}
	m.GetLogger(DEFAULT).Info("connected to %s", "postgres")
	m.Close()
	if !strings.Contains(out.String(), "\"message\":\"connected to postgres\"") {
		t.Fatalf("unexpected logging: %s", out.String())
	}
}
func TestFileWriterIsClosedAndRetainsEvents(t *testing.T) {
	dir := t.TempDir()
	m := NewManager()
	if err := m.ConfigureOutputs("file", "json", "info", dir, "test.log"); err != nil {
		t.Fatal(err)
	}
	m.GetLogger(DEFAULT).Info("worker ready")
	m.Close()
	data, err := os.ReadFile(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("worker ready")) {
		t.Fatalf("missing event: %s", data)
	}
	m.Close()
}
func TestInvalidWriterRejected(t *testing.T) {
	m := NewManager()
	defer m.Close()
	if err := m.ConfigureOutputs("console,unknown", "json", "info", t.TempDir(), "test.log"); err == nil {
		t.Fatal("unsupported writer was accepted")
	}
}
