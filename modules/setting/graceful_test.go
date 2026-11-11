package setting

import (
	"testing"
	"time"
)

func TestGracefulSettingsDefaultsAndOverrides(t *testing.T) {
	provider, err := NewConfigProviderFromData("[server]\n")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := loadGracefulSettingsFrom(provider)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPShutdownTimeout != 10*time.Second || cfg.HammerTimeout != 15*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	provider, err = NewConfigProviderFromData("[server]\nHTTP_SHUTDOWN_TIMEOUT = 7s\nGRACEFUL_HAMMER_TIMEOUT = 12s\n")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = loadGracefulSettingsFrom(provider)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPShutdownTimeout != 7*time.Second || cfg.HammerTimeout != 12*time.Second {
		t.Fatalf("unexpected overrides: %+v", cfg)
	}
}

func TestGracefulSettingsRejectsInvalidTimeouts(t *testing.T) {
	for _, data := range []string{
		"[server]\nHTTP_SHUTDOWN_TIMEOUT = invalid\n",
		"[server]\nHTTP_SHUTDOWN_TIMEOUT = 0s\n",
		"[server]\nHTTP_SHUTDOWN_TIMEOUT = 15s\nGRACEFUL_HAMMER_TIMEOUT = 10s\n",
		"[server]\nGRACEFUL_HAMMER_TIMEOUT = -1s\n",
	} {
		provider, err := NewConfigProviderFromData(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadGracefulSettingsFrom(provider); err == nil {
			t.Fatalf("accepted invalid graceful settings %q", data)
		}
	}
}
