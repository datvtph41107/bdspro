package setting

import "testing"

func TestLoadDBSetting(t *testing.T) {
	provider, err := NewConfigProviderFromData("[database]\nDB_SOURCE = postgres://localhost/bdspro\nMIGRATION_URL = file://db/migration\n")
	if err != nil {
		t.Fatal(err)
	}
	loadDBSetting(provider)
	if Database.DBSource != "postgres://localhost/bdspro" {
		t.Fatalf("DB source = %q", Database.DBSource)
	}
	if Database.MigrationURL != "file://db/migration" {
		t.Fatalf("migration URL = %q", Database.MigrationURL)
	}
}
