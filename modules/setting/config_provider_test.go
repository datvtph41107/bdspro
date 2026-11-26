package setting

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigProviderFileAndDataParity(t *testing.T) {
	content := "RUN_MODE = development\n\n[log]\nMODE = console\nEMPTY =\n\n[log.file]\nFILE_NAME = trace.log\n"
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	fromFile, err := NewConfigProviderFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	fromData, err := NewConfigProviderFromData(content)
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range []ConfigProvider{fromFile, fromData} {
		if v := provider.Section("").Key("RUN_MODE").String(); v != "development" {
			t.Fatalf("RUN_MODE=%q", v)
		}
		if v := provider.Section("log.file").Key("FILE_NAME").String(); v != "trace.log" {
			t.Fatalf("log.file.FILE_NAME=%q", v)
		}
		if !provider.Section("log").Key("EMPTY").IsSet() {
			t.Fatal("explicit empty key not recognized")
		}
		if provider.Section("log").Key("MISSING").IsSet() {
			t.Fatal("missing key incorrectly marked as set")
		}
		if provider.Section("log.file").Key("MODE").IsSet() {
			t.Fatal("child section inherited parent MODE")
		}
	}
}

func TestConfigProviderRejectsMissingFile(t *testing.T) {
	_, err := NewConfigProviderFromFile(filepath.Join(t.TempDir(), "missing.ini"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want os.ErrNotExist, got %v", err)
	}
}

func TestConfigProviderRejectsEmptyFilename(t *testing.T) {
	if _, err := NewConfigProviderFromFile(""); err == nil {
		t.Fatal("empty filename should fail")
	}
}
