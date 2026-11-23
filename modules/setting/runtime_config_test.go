package setting

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

const validWebFixture = `RUN_MODE = development
[server]
HTTP_SERVER_ADDRESS = :9090
[security]
TOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901
ACCESS_TOKEN_DURATION = 15m
REFRESH_TOKEN_DURATION = 24h
[database]
DB_SOURCE = postgres://localhost/first
MIGRATION_URL = file://db/migration
[redis]
ADDR = localhost:6379
`

func TestCheckWebConfigDoesNotPublishProcessState(t *testing.T) {
	saveGlobals(t)
	AppWorkPath = "before-work"
	Server = ServerSettings{HTTPServerAddress: "before-listener"}
	file := writeConfig(t, validWebFixture)
	if err := CheckWebConfig(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	if AppWorkPath != "before-work" || Server.HTTPServerAddress != "before-listener" {
		t.Fatal("config check changed process globals")
	}
}

func TestBuildMigrateSettingsDoesNotRequireWebAuthOrRedis(t *testing.T) {
	saveGlobals(t)
	file := writeConfig(t, "RUN_MODE = development\n[database]\nDB_SOURCE = postgres://localhost/maintenance\nMIGRATION_URL = file://db/migration\n")
	state, err := buildMigrateSettings(PathOptions{Config: file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "file://" + filepath.ToSlash(filepath.Join(state.paths.WorkPath, "db", "migration"))
	if state.database.MigrationURL != want || state.database.DBSource != "postgres://localhost/maintenance" {
		t.Fatalf("unexpected migration settings: %+v", state.database)
	}
	if Database.DBSource != "" {
		t.Fatal("builder published a database global")
	}
}

func TestCanonicalWebSettingsCopyMutablePolicyMap(t *testing.T) {
	saveGlobals(t)
	policies := ipactive.DefaultPolicies()
	state := startupSettings{ipactive: IPActiveSettings{Policies: policies}}
	if err := commitSettings(state, true); err != nil {
		t.Fatal(err)
	}
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

func TestBuildWebSettingsFailurePreservesPublishedSettings(t *testing.T) {
	saveGlobals(t)
	Server = ServerSettings{HTTPServerAddress: "unchanged"}
	file := writeConfig(t, "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = wrong\n")
	err := CheckWebConfig(PathOptions{Config: file}, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP_SERVER_ADDRESS") {
		t.Fatalf("expected listener validation error, got %v", err)
	}
	if Server.HTTPServerAddress != "unchanged" {
		t.Fatal("failed check mutated published settings")
	}
}

func TestOnlyOneRuntimeSettingsPublicationPerProcess(t *testing.T) {
	saveGlobals(t)
	file := writeConfig(t, validWebFixture)
	if err := LoadWebSettings(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	if err := LoadWebSettings(PathOptions{Config: file}, nil); err == nil ||
		!strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("second publication must fail, got %v", err)
	}
	if Server.HTTPServerAddress != ":9090" || Database.DBSource != "postgres://localhost/first" {
		t.Fatal("second publication changed process state")
	}
	if err := CheckWebConfig(PathOptions{Config: file}, nil); err != nil {
		t.Fatalf("pure config check should still work: %v", err)
	}
}

func TestMigrateSettingsDoesNotPublishWebSubsystems(t *testing.T) {
	saveGlobals(t)
	Server = ServerSettings{HTTPServerAddress: "keep-http"}
	Security = SecuritySettings{TokenSymmetricKey: "keep-auth"}
	Redis = RedisSettings{Address: "keep-redis"}
	file := writeConfig(t, "RUN_MODE = development\n[database]\nDB_SOURCE = postgres://localhost/migrate\nMIGRATION_URL = file://db/migration\n")
	if err := LoadMigrateSettings(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	if Database.DBSource != "postgres://localhost/migrate" || RunMode != "development" {
		t.Fatal("migrate did not publish canonical DB settings")
	}
	if Server.HTTPServerAddress != "keep-http" || Security.TokenSymmetricKey != "keep-auth" || Redis.Address != "keep-redis" {
		t.Fatal("migrate unexpectedly mutated Web/Auth/Redis settings")
	}
}

func TestCheckWebRejectsMissingResourceConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  string
	}{
		{"database", strings.Replace(validWebFixture, "DB_SOURCE = postgres://localhost/first", "DB_SOURCE = ", 1)},
		{"redis", strings.Replace(validWebFixture, "ADDR = localhost:6379", "ADDR = ", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := writeConfig(t, tc.cfg)
			if err := CheckWebConfig(PathOptions{Config: file}, nil); err == nil {
				t.Fatal("missing required connection configuration was accepted")
			}
		})
	}
}
