package setting

import (
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func TestLoadCacheSettingsDefaults(t *testing.T) {
	f, err := ini.Load([]byte("[database]\nDB_SOURCE=unused\n"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadCacheSettings(&ConfigProvider{file: f})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Adapter != "memory" || cfg.Prefix != "cache:" || cfg.MaxEntries != 10000 || cfg.DefaultTTL != 5*time.Minute {
		t.Fatalf("unexpected cache defaults: %+v", cfg)
	}
}

func TestLoadCacheSettingsRejectsInvalidConfiguration(t *testing.T) {
	for _, section := range []string{
		"[cache]\nADAPTER=invalid\n",
		"[cache]\nMAX_ENTRIES=0\n",
		"[cache]\nITEM_TTL=0s\n",
		"[cache]\nADAPTER=memcache\n",
	} {
		f, err := ini.Load([]byte(section))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := LoadCacheSettings(&ConfigProvider{file: f}); err == nil {
			t.Fatalf("invalid config accepted: %q", section)
		}
	}
}
