package setting

import (
	"path/filepath"
	"testing"
)

// Both deployed first-boot and local templates must advertise exactly the
// OAuth2 keys consumed by modules/setting; stale provider examples otherwise
// create configuration drift without failing CI or initial server startup.
func TestCommittedConfigTemplatesUseOAuth2Provider(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "local example", path: filepath.Join("..", "..", "custom", "conf", "app.example.ini")},
		{name: "Docker first boot", path: filepath.Join("..", "..", "docker", "root", "etc", "templates", "app.ini")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := InitCfgProvider(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			if !provider.file.HasSection("oauth2") {
				t.Fatal("template is missing current [oauth2] configuration section")
			}
			if provider.file.HasSection("oauth") {
				t.Fatal("template still documents obsolete [oauth] provider")
			}
			providerSection := provider.file.Section("oauth2")
			for _, key := range []string{
				"ISSUER", "AUTHORIZATION_URL", "TOKEN_URL", "JWKS_URL",
				"CLIENT_ID", "CLIENT_SECRET", "REDIRECT_URI",
			} {
				if !providerSection.HasKey(key) {
					t.Fatalf("missing OAuth2 config key %q", key)
				}
			}
			if _, err := LoadSettings(provider); err != nil {
				t.Fatalf("first-boot template must load under current settings: %v", err)
			}
		})
	}
}
