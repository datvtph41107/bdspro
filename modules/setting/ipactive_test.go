package setting

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

func TestIPActiveConfigurationOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.ini")
	ini := "[ipactive]\nREGISTER_MODE = cooldown\nREGISTER_LIMIT = 1\nREGISTER_WINDOW = 15s\nLOGIN_LIMIT = 7\n"
	if err := os.WriteFile(path, []byte(ini), 0600); err != nil {
		t.Fatal(err)
	}
	provider, err := InitCfgProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	policies, err := LoadIPActivePolicies(provider)
	if err != nil {
		t.Fatal(err)
	}
	if got := policies[ipactive.Register]; got != (ipactive.Policy{Mode: ipactive.ModeCooldown, Limit: 1, Window: 15 * time.Second}) {
		t.Fatalf("incorrect cooldown override: %+v", got)
	}
	if policies[ipactive.Login].Limit != 7 || policies[ipactive.Login].Window != time.Minute {
		t.Fatalf("unexpected login policy: %+v", policies[ipactive.Login])
	}
	if policies[ipactive.RenewAccess] != ipactive.DefaultPolicies()[ipactive.RenewAccess] {
		t.Fatal("unspecified policies were not preserved")
	}
}

func TestIPActiveUnsafeConfigurationRejected(t *testing.T) {
	for _, entry := range []string{
		"REGISTER_LIMIT = 0",
		"REGISTER_WINDOW = 0s",
		"REGISTER_MODE = unknown",
		"REGISTER_MODE = cooldown",
	} {
		path := filepath.Join(t.TempDir(), "app.ini")
		if err := os.WriteFile(path, []byte("[ipactive]\n"+entry+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		provider, err := InitCfgProvider(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := LoadIPActivePolicies(provider); err == nil {
			t.Fatalf("unsafe configuration accepted: %q", entry)
		}
	}
}
