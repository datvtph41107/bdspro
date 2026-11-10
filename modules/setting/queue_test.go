package setting

import (
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

func TestQueueSettingsDefaultsAndOverrides(t *testing.T) {
	f, err := ini.Load([]byte("[queue]\nCRITICAL_WEIGHT=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadQueueSettings(&ConfigProvider{file: f})
	if err != nil {
		t.Fatal(err)
	}
	if got.Concurrency != 10 || got.CriticalWeight != 2 || got.DefaultWeight != 5 {
		t.Fatalf("unexpected mixed queue settings: %+v", got)
	}
}

func TestQueueSettingsRequireProvider(t *testing.T) {
	if _, err := LoadQueueSettings(nil); err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("nil queue config provider accepted: %v", err)
	}
}

func TestQueueSettingsRejectsExplicitEmptyOverrides(t *testing.T) {
	for _, tc := range []struct {
		name       string
		config     string
		emptyEnv   bool
		wantOption string
	}{
		{"empty_ini_concurrency", "[queue]\nCONCURRENCY =\n", false, "CONCURRENCY"},
		{"empty_ini_weight", "[queue]\nDEFAULT_WEIGHT =\n", false, "DEFAULT_WEIGHT"},
		{"empty_env_concurrency", "[queue]\nCONCURRENCY = 3\n", true, "CONCURRENCY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := ini.Load([]byte(tc.config))
			if err != nil {
				t.Fatal(err)
			}
			if tc.emptyEnv {
				t.Setenv("BDS__QUEUE__CONCURRENCY", "")
			}
			if _, err := LoadQueueSettings(&ConfigProvider{file: f}); err == nil || !strings.Contains(err.Error(), tc.wantOption) {
				t.Fatalf("explicit empty %s accepted: %v", tc.wantOption, err)
			}
		})
	}
}

func TestQueueSettingsMissingOverridesRetainDefaults(t *testing.T) {
	f, err := ini.Load([]byte("[queue]\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadQueueSettings(&ConfigProvider{file: f})
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultQueueSettings() {
		t.Fatalf("missing queue options changed defaults: %+v", got)
	}
}

func TestQueueSettingsRejectsResourceExhaustionOptions(t *testing.T) {
	for _, tc := range []struct {
		name, option, want string
	}{
		{"too_many_workers", "CONCURRENCY = 257", "CONCURRENCY"},
		{"critical_weight_bomb", "CRITICAL_WEIGHT = 1000000000", "CRITICAL_WEIGHT"},
		{"default_weight_bomb", "DEFAULT_WEIGHT = 101", "DEFAULT_WEIGHT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := ini.Load([]byte("[queue]\n" + tc.option + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := LoadQueueSettings(&ConfigProvider{file: f}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("resource-exhausting %q accepted: %v", tc.option, err)
			}
		})
	}
}

func TestQueueSettingsAcceptsResourceBounds(t *testing.T) {
	f, err := ini.Load([]byte("[queue]\nCONCURRENCY = 256\nCRITICAL_WEIGHT = 100\nDEFAULT_WEIGHT = 100\n"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadQueueSettings(&ConfigProvider{file: f})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Concurrency != MaxQueueConcurrency || cfg.CriticalWeight != MaxQueueWeight || cfg.DefaultWeight != MaxQueueWeight {
		t.Fatalf("resource-bound configuration not preserved: %+v", cfg)
	}
}
