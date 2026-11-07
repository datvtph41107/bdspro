package setting

import "testing"

func TestBrowserPublicOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin, environment string
		ok                  bool
	}{
		{"", "production", true},
		{"https://auth.example.test", "production", true},
		{"https://auth.example.test:8443", "production", true},
		{"http://localhost:8080", "development", true},
		{"http://127.0.0.1:8080", "development", true},
		{"http://auth.example.test", "development", false},
		{"http://localhost:8080", "production", false},
		{"https://auth.example.test/path", "production", false},
		{"https://auth.example.test?x=1", "production", false},
		{"https://u@auth.example.test", "production", false},
		{"javascript:alert(1)", "production", false},
		{"//attacker.example", "production", false},
	} {
		err := ValidateBrowserPublicOrigin(tc.origin, tc.environment)
		if (err == nil) != tc.ok {
			t.Errorf("origin=%q env=%q err=%v, want ok=%v", tc.origin, tc.environment, err, tc.ok)
		}
	}
}
