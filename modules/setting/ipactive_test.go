package setting

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

func TestIPActiveAbsentKeysKeepDefaults(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[ipactive]\n")
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadIPActiveFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defaults := ipactive.DefaultPolicies()
	if len(got.Policies) != len(defaults) {
		t.Fatalf("policy count = %d, want %d", len(got.Policies), len(defaults))
	}
	for action, want := range defaults {
		if value := got.Policies[action]; value != want {
			t.Fatalf("%s: default policy = %+v, want %+v", action, value, want)
		}
	}
}

func TestIPActiveExplicitEmptyOverridesRejected(t *testing.T) {
	for action := range ipactive.DefaultPolicies() {
		for _, suffix := range []string{"LIMIT", "WINDOW", "MODE"} {
			name := strings.ToUpper(string(action)) + "_" + suffix
			t.Run(name, func(t *testing.T) {
				cfg, err := NewConfigProviderFromData(fmt.Sprintf("[ipactive]\n%s =\n", name))
				if err != nil {
					t.Fatal(err)
				}
				if err := validateKnownKeys(cfg); err != nil {
					t.Fatalf("schema rejected known key %s: %v", name, err)
				}
				_, err = loadIPActiveFrom(cfg)
				if err == nil || !strings.Contains(err.Error(), "ipactive."+name) {
					t.Fatalf("%s: want named error for explicit empty, got %v", name, err)
				}
			})
		}
	}
}

func TestIPActiveValidOverridesKeepOtherDefaults(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[ipactive]\nLOGIN_LIMIT = 1\nLOGIN_WINDOW = 2m\nLOGIN_MODE = cooldown\n")
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadIPActiveFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := ipactive.Policy{Mode: ipactive.ModeCooldown, Limit: 1, Window: 2 * time.Minute}
	if actual := got.Policies[ipactive.Login]; actual != want {
		t.Fatalf("LOGIN policy = %+v, want %+v", actual, want)
	}
	if actual := got.Policies[ipactive.CreateTeam]; actual != ipactive.DefaultPolicies()[ipactive.CreateTeam] {
		t.Fatalf("unrelated policy changed: %+v", actual)
	}
}

func TestIPActiveEmptyEnvironmentOverrideFailsConfigCheck(t *testing.T) {
	saveGlobals(t)
	file := writeConfig(t, validWebFixture)
	err := CheckWebConfig(PathOptions{Config: file}, []string{"BDS__IPACTIVE__LOGIN_LIMIT="})
	if err == nil || !strings.Contains(err.Error(), "ipactive.LOGIN_LIMIT") {
		t.Fatalf("empty IPActive ENV override must fail preflight, got %v", err)
	}
}
