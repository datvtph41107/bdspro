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
