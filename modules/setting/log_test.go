package setting

import (
	"testing"
)

func TestLogOutputConfiguration(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[log]\nMODE=console,file\nFORMAT=console\n[log.file]\nFORMAT=json\nMAX_SIZE_MB=10\nMAX_BACKUPS=3\nMAX_AGE_DAYS=7\n")
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadLogGlobalFrom(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got.ConsoleFormat != "console" || got.FileFormat != "json" || got.MaxSizeMB != 10 {
		t.Fatalf("wrong outputs: %+v", got)
	}
	for _, text := range []string{"[log.file]\nMAX_SIZE_MB=0", "[log.file]\nMAX_BACKUPS=-1", "[log.file]\nFORMAT=xml", "[log.file]\nMAX_AGE_DAYS=oops"} {
		cfg, _ := NewConfigProviderFromData(text)
		if _, err := loadLogGlobalFrom(cfg, t.TempDir()); err == nil {
			t.Fatalf("invalid config accepted: %s", text)
		}
	}
}

func TestSQLLogConfiguration(t *testing.T) {
	for _, text := range []string{"[database]\nLOG_SQL=verbose", "[database]\nSLOW_QUERY_THRESHOLD=0", "[database]\nSLOW_QUERY_THRESHOLD=-1s"} {
		cfg, _ := NewConfigProviderFromData(text)
		if _, err := loadDBSetting(cfg, t.TempDir()); err == nil {
			t.Fatalf("invalid SQL config accepted: %s", text)
		}
	}
}

func TestLogDestinationPolicy(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[log]\nLEVEL=info\n[log.console]\nLEVEL=warn\nCOMPONENTS=http, sql\n[log.file]\nLEVEL=debug\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateKnownKeys(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := loadLogGlobalFrom(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	o := got.OutputOptions()
	if o.ConsoleLevel != "warn" || o.FileLevel != "debug" || o.ConsoleComponents != "http, sql" {
		t.Fatalf("wrong destination policy: %+v", o)
	}
	inherited, _ := NewConfigProviderFromData("[log]\nLEVEL=error\n")
	got, err = loadLogGlobalFrom(inherited, t.TempDir())
	if err != nil || got.ConsoleLevel != "error" || got.FileLevel != "error" {
		t.Fatalf("wrong inherited policy: %+v %v", got, err)
	}
	for _, invalid := range []string{"[log.console]\nLEVEL=verbose", "[log.file]\nLEVEL=trace", "[log.console]\nCOMPONENTS=http,,sql"} {
		cfg, err := NewConfigProviderFromData(invalid)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadLogGlobalFrom(cfg, t.TempDir()); err == nil {
			t.Fatalf("invalid policy accepted: %s", invalid)
		}
	}
}
