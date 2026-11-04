package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectTypes(t *testing.T) {
	dir := t.TempDir()
	source := "package sqlc\ntype Account struct{}\ntype hidden struct{}\ntype Queries struct{}\n"
	if err := os.WriteFile(filepath.Join(dir, "db.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	names, err := collectTypes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Account", "Queries"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("types = %v, want %v", names, want)
	}
}
