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
