package setting

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseTrustedProxies(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want []string
	}{
		{raw: "", want: nil},
		{raw: "  ", want: nil},
		{raw: "192.0.2.1", want: []string{"192.0.2.1"}},
		{raw: "192.0.2.1, 198.51.100.0/28, 2001:db8::/64",
			want: []string{"192.0.2.1", "198.51.100.0/28", "2001:db8::/64"}},
	} {
		got, err := ParseTrustedProxies(tc.raw)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ParseTrustedProxies(%q) = %v, %v; want %v", tc.raw, got, err, tc.want)
		}
	}
}

func TestParseTrustedProxiesFailClosed(t *testing.T) {
	for _, raw := range []string{
		",", "192.0.2.1,", ",192.0.2.1", "192.0.2.1,,198.51.100.1",
		"proxy.internal", "0.0.0.0/0", "192.0.2.1/0", "::/0",
		"::ffff:0:0/96", "fe80::1%eth0", "invalid",
	} {
		if _, err := ParseTrustedProxies(raw); err == nil {
			t.Fatalf("unsafe TRUSTED_PROXIES accepted: %q", raw)
		}
	}
	if err := ValidateTrustedProxies([]string{" 192.0.2.1"}); err == nil {
		t.Fatal("manually built unsafe config accepted")
	}
}

func TestTrustedProxiesConfigurationOverride(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app.ini")
	data := strings.Join([]string{
		"RUN_MODE = development",
		"[server]",
		"HTTP_SERVER_ADDRESS = 127.0.0.1:8080",
		"TRUSTED_PROXIES = 192.0.2.1",
		"[database]",
		"DB_SOURCE = postgres://test",
		"MIGRATION_URL = file://db/migration",
		"[security]",
		"TOKEN_SYMMETRIC_KEY = 01234567890123456789012345678901",
		"ACCESS_TOKEN_DURATION = 15m",
		"REFRESH_TOKEN_DURATION = 24h",
	}, "\n") + "\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BDS__SERVER__TRUSTED_PROXIES", "198.51.100.10/32")
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.TrustedProxies, []string{"198.51.100.10/32"}) {
		t.Fatalf("unexpected environment override: %v", cfg.TrustedProxies)
	}
	t.Setenv("BDS__SERVER__TRUSTED_PROXIES", "::/0")
	if _, err := LoadConfig(file); err == nil {
		t.Fatal("dangerous environment override accepted")
	}
}
