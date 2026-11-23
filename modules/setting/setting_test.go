package setting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// saveGlobals ngăn tests làm bẩn state lẫn nhau; tests không chạy parallel.
func saveGlobals(t *testing.T) {
	t.Helper()
	oldWork, oldCustom, oldConf := AppWorkPath, CustomPath, CustomConf
	oldRunMode, oldLog, oldGraceful := RunMode, Log, Graceful
	settingsInit.Lock()
	wasPublished := settingsInit.done
	settingsInit.Unlock()
	oldServer, oldAPI, oldSecurity, oldIP := Server, API, Security, IPActive
	oldDB, oldRedis := Database, Redis
	t.Cleanup(func() {
		settingsInit.Lock()
		defer settingsInit.Unlock()
		AppWorkPath, CustomPath, CustomConf = oldWork, oldCustom, oldConf
		RunMode, Log, Graceful = oldRunMode, oldLog, oldGraceful
		settingsInit.done = wasPublished
		Server, API, Security, IPActive = oldServer, oldAPI, oldSecurity, oldIP
		Database, Redis = oldDB, oldRedis
	})
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "app.ini")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestWorkPathFromINIAndPublication(t *testing.T) {
	saveGlobals(t)
	work := t.TempDir()
	file := writeConfig(t, "RUN_MODE = development\nWORK_PATH = "+work+"\n[log]\nROOT_PATH = data/log\n[database]\nDB_SOURCE = example\n")
	if err := LoadMigrateSettings(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	if AppWorkPath != work || CustomPath != filepath.Join(work, "custom") || CustomConf != file {
		t.Fatalf("published paths: work=%q custom=%q conf=%q", AppWorkPath, CustomPath, CustomConf)
	}
	if got := Log.RootPath; got != filepath.Join(work, "data/log") {
		t.Fatalf("Log.RootPath=%q", got)
	}
}

func TestWorkPathConflictDoesNotPublish(t *testing.T) {
	saveGlobals(t)
	AppWorkPath, CustomPath, CustomConf = "previous-work", "previous-custom", "previous-conf"
	work := t.TempDir()
	file := writeConfig(t, "WORK_PATH = "+work+"\n")
	different := t.TempDir()
	err := LoadMigrateSettings(PathOptions{WorkPath: different, Config: file}, nil)
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("want conflict, got %v", err)
	}
	if AppWorkPath != "previous-work" || CustomPath != "previous-custom" || CustomConf != "previous-conf" {
		t.Fatal("globals changed on conflict")
	}
}

func TestEnvironmentWorkPathAndInvalidConfig(t *testing.T) {
	saveGlobals(t)
	work := t.TempDir()
	file := writeConfig(t, "RUN_MODE = development\n[database]\nDB_SOURCE = example\n")
	if err := LoadMigrateSettings(PathOptions{Config: file}, []string{"BDS____WORK_PATH=" + work}); err != nil {
		t.Fatal(err)
	}
	if AppWorkPath != work {
		t.Fatalf("environment WORK_PATH=%q", AppWorkPath)
	}
	prev := AppWorkPath
	if err := LoadMigrateSettings(PathOptions{Config: file}, []string{"BDS____RUN_MODE=invalid"}); err == nil {
		t.Fatal("invalid run mode accepted")
	}
	if AppWorkPath != prev {
		t.Fatal("globals changed on invalid RUN_MODE")
	}
}

func TestLoadWebSettingsNoPartialPublication(t *testing.T) {
	saveGlobals(t)
	Server = ServerSettings{HTTPServerAddress: "old"}
	file := writeConfig(t, "RUN_MODE = development\n[server]\nHTTP_SERVER_ADDRESS = 127.0.0.1:8080\n[api]\nMAX_RESPONSE_ITEMS = 1\nDEFAULT_PAGING_NUM = 2\n")
	if err := LoadWebSettings(PathOptions{Config: file}, nil); err == nil {
		t.Fatal("invalid API configuration accepted")
	}
	if Server.HTTPServerAddress != "old" || settingsInit.done {
		t.Fatal("Web runtime settings published before full validation")
	}
}

func TestLoadWebSettingsSuccess(t *testing.T) {
	saveGlobals(t)
	const cfg = `RUN_MODE = development
[server]
HTTP_SERVER_ADDRESS = 127.0.0.1:8080
[security]
TOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901
ACCESS_TOKEN_DURATION = 15m
REFRESH_TOKEN_DURATION = 24h
[database]
DB_SOURCE = postgres://localhost/db
[redis]
ADDR = 127.0.0.1:6379
`
	file := writeConfig(t, cfg)
	if err := LoadWebSettings(PathOptions{Config: file}, nil); err != nil {
		t.Fatal(err)
	}
	if Server.HTTPServerAddress != "127.0.0.1:8080" || Database.DBSource != "postgres://localhost/db" || Redis.Address != "127.0.0.1:6379" {
		t.Fatalf("wrong settings: server=%+v database=%+v redis=%+v", Server, Database, Redis)
	}
}

func TestExplicitEmptyRunModeIsRejected(t *testing.T) {
	cfg, err := NewConfigProviderFromData("RUN_MODE =\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadRunModeFrom(cfg); err == nil {
		t.Fatal("RUN_MODE explicitly empty must not become development")
	}
}

func TestExplicitEmptyLogModeIsRejected(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[log]\nMODE =\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadLogGlobalFrom(cfg, t.TempDir()); err == nil {
		t.Fatal("log.MODE explicitly empty must not become console")
	}
}
