package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExist(t *testing.T) {
	dir := t.TempDir()
	if ok, err := IsExist(filepath.Join(dir, "missing")); ok || err != nil {
		t.Fatalf("missing: exists=%v, err=%v", ok, err)
	}
	file := filepath.Join(dir, "present")
	if err := os.WriteFile(file, []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if ok, err := IsExist(file); !ok || err != nil {
		t.Fatalf("present: exists=%v, err=%v", ok, err)
	}
}
