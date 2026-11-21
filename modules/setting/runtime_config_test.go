package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

func TestLoadWebConfigDoesNotPublishProcessState(t *testing.T) {
	saveGlobals(t)
	AppWorkPath = "before-work"
	Server = ServerSettings{HTTPServerAddress: "before-listener"}
	CfgProvider = nil
	content := "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n" +
		"[security]\nTOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901\n" +
		"ACCESS_TOKEN_DURATION = 15m\nREFRESH_TOKEN_DURATION = 24h\n" +
		"[database]\nDB_SOURCE = postgres://localhost/dev\n[redis]\nADDR = localhost:6379\n"
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadWebConfig(PathOptions{Config: file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DBSource != "postgres://localhost/dev" || cfg.Server.HTTPServerAddress != "127.0.0.1:8080" {
		t.Fatal("invalid typed Web snapshot")
	}
	if AppWorkPath != "before-work" || Server.HTTPServerAddress != "before-listener" || CfgProvider != nil {
		t.Fatal("LoadWebConfig changed process state")
	}
}

func TestLoadMigrateConfigDoesNotRequireWebAuthOrRedis(t *testing.T) {
	saveGlobals(t)
	content := "RUN_MODE = development\n[database]\n" +
		"DB_SOURCE = postgres://localhost/maintenance\n" +
		"MIGRATION_URL = file://db/migration\n"
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadMigrateConfig(PathOptions{Config: file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "file://" + filepath.ToSlash(filepath.Join(cfg.Paths.WorkPath, "db", "migration"))
	if cfg.Database.MigrationURL != want || cfg.Database.DBSource != "postgres://localhost/maintenance" {
		t.Fatalf("wrong migration configuration: %+v", cfg.Database)
	}
	if Database.DBSource != "" {
		t.Fatal("migration loader published a database global")
	}
}

func TestPublishLegacyWebCopiesMutablePolicyMap(t *testing.T) {
	saveGlobals(t)
	policies := ipactive.DefaultPolicies()
	cfg := WebConfig{IPActive: IPActiveSettings{Policies: policies}}
	PublishLegacyWeb(cfg)
	if len(policies) == 0 {
		t.Fatal("expected default IP policy")
	}
	for action := range policies {
		old := IPActive.Policies[action]
		policies[action] = ipactive.Policy{}
		if IPActive.Policies[action] != old {
			t.Fatal("published global shares policy map with caller")
		}
		break
	}
}

func TestLoadWebConfigFailurePreservesPublishedSettings(t *testing.T) {
	saveGlobals(t)
	Server = ServerSettings{HTTPServerAddress: "unchanged"}
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte("RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = wrong\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadWebConfig(PathOptions{Config: file}, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP_SERVER_ADDRESS") {
		t.Fatalf("expected listener validation error, got %v", err)
	}
	if Server.HTTPServerAddress != "unchanged" {
		t.Fatal("failed load mutated published settings")
	}
}
