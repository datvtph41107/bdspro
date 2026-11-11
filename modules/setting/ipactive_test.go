package setting

import (
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

func TestIPActiveConfigurationOverrides(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[ipactive]\nREGISTER_MODE = cooldown\nREGISTER_LIMIT = 1\nREGISTER_WINDOW = 15s\nLOGIN_LIMIT = 7\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := loadIPActiveFrom(cfg); err != nil {
		t.Fatal(err)
	}
	if got := IPActive.Policies[ipactive.Register]; got != (ipactive.Policy{Mode: ipactive.ModeCooldown, Limit: 1, Window: 15 * time.Second}) {
		t.Fatalf("register policy = %+v", got)
	}
	if got := IPActive.Policies[ipactive.Login]; got.Limit != 7 || got.Window != time.Minute {
		t.Fatalf("login policy = %+v", got)
	}
}

func TestIPActiveUnsafeConfigurationRejected(t *testing.T) {
	for _, entry := range []string{"REGISTER_LIMIT = 0", "REGISTER_WINDOW = 0s", "REGISTER_MODE = unknown", "REGISTER_MODE = cooldown"} {
		cfg, err := NewConfigProviderFromData("[ipactive]\n" + entry + "\n")
		if err != nil {
			t.Fatal(err)
		}
		if err := loadIPActiveFrom(cfg); err == nil {
			t.Errorf("unsafe IPActive override accepted: %s", entry)
		}
	}
}
