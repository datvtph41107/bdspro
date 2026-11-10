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

func TestFileLoggerCreatesMissingDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data", "log")
	m := NewManager()
	if err := m.ConfigureOutputs("file", "json", "info", root, "worker.log"); err != nil {
		t.Fatal(err)
	}
	m.GetLogger(DEFAULT).Info("created on first startup")
	m.Close()

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Fatalf("log directory not prepared: stat=%v err=%v", info, err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("new log directory grants group/other access: %v", info.Mode().Perm())
	}
	data, err := os.ReadFile(filepath.Join(root, "worker.log"))
	if err != nil || !bytes.Contains(data, []byte("created on first startup")) {
		t.Fatalf("file sink not writable after creating directory: data=%q err=%v", data, err)
	}
}

func TestFileLoggerRejectsUnusableLogRoot(t *testing.T) {
	parent := t.TempDir()
	file := filepath.Join(parent, "regular-file")
	if err := os.WriteFile(file, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	defer m.Close()
	if err := m.ConfigureOutputs("file", "json", "info", filepath.Join(file, "log"), "worker.log"); err == nil {
		t.Fatal("file logger accepted an unusable root directory")
	}
}

func TestConsoleLoggerDoesNotCreateUnusedLogDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data", "log")
	m := NewManager()
	if err := m.ConfigureOutputs("console", "json", "info", root, "worker.log"); err != nil {
		t.Fatal(err)
	}
	m.Close()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("console-only logger created log directory or unexpected stat error: %v", err)
	}
}
