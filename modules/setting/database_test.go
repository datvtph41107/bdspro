package setting

import (
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

func TestLoadDBSettingUsesAppWorkPath(t *testing.T) {
	saveGlobals(t)
	provider, err := NewConfigProviderFromData("[database]\nDB_SOURCE = example\nMIGRATION_URL = file://db/migration\n")
	if err != nil {
		t.Fatal(err)
	}
	AppWorkPath = t.TempDir()
	CfgProvider = provider
	if err := LoadDBSetting(); err != nil {
		t.Fatal(err)
	}
	want := "file://" + filepath.ToSlash(filepath.Join(AppWorkPath, "db", "migration"))
	if Database.MigrationURL != want {
		t.Fatalf("migration path = %q, want %q", Database.MigrationURL, want)
	}
}

func TestLoadDBSettingRejectsInvalidWorkPathWithoutPublication(t *testing.T) {
	saveGlobals(t)
	provider, err := NewConfigProviderFromData("[database]\nMIGRATION_URL = file://db/migration\n")
	if err != nil {
		t.Fatal(err)
	}
	CfgProvider = provider
	AppWorkPath = ""
	Database = DatabaseSettings{MigrationURL: "previous"}
	if err := LoadDBSetting(); err == nil {
		t.Fatalf("expected work path error, got %v", err)
	}
	if Database.MigrationURL != "previous" {
		t.Fatal("migration settings published on error")
	}
}
