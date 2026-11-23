package setting

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMigrationURL(t *testing.T) {
	const work = "/srv/bdspro"
	cases := []struct {
		raw, want string
	}{
		{"file://db/migration", "file:///srv/bdspro/db/migration"},
		{"file:///opt/migrations", "file:///opt/migrations"},
		{"postgres://hostname/migration", "postgres://hostname/migration"},
		{"", ""},
	}
	for _, tt := range cases {
		got, err := resolveMigrationURL(tt.raw, work)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("resolveMigrationURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestMigrateSettingsUsesResolvedWorkPath(t *testing.T) {
	saveGlobals(t)
	work := t.TempDir()
	file := filepath.Join(work, "app.ini")
	content := "RUN_MODE = development\nWORK_PATH = " + work + "\n" +
		"[database]\nDB_SOURCE = example\nMIGRATION_URL = file://db/migration\n"
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadMigrateSettings(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	want := "file://" + filepath.ToSlash(filepath.Join(work, "db", "migration"))
	if Database.MigrationURL != want {
		t.Fatalf("migration path = %q, want %q", Database.MigrationURL, want)
	}
}

func TestMigrateLoaderRejectsInvalidWorkPathWithoutPublication(t *testing.T) {
	saveGlobals(t)
	provider, err := NewConfigProviderFromData("[database]\nDB_SOURCE = example\nMIGRATION_URL = file://db/migration\n")
	if err != nil {
		t.Fatal(err)
	}
	Database = DatabaseSettings{MigrationURL: "previous"}
	if _, err := loadDBSetting(provider, ""); err == nil {
		t.Fatal("expected work path error")
	}
	if Database.MigrationURL != "previous" {
		t.Fatal("migration settings published on error")
	}
}
