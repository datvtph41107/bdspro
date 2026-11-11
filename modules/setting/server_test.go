package setting

import "testing"

func TestTrustedProxies(t *testing.T) {
	peers, err := ParseTrustedProxies("127.0.0.1,10.0.0.0/8")
	if err != nil || len(peers) != 2 {
		t.Fatalf("trusted proxies = %v, %v", peers, err)
	}
	for _, raw := range []string{"0.0.0.0/0", "::/0", "example.com", "10.0.0.0/8,"} {
		if _, err := ParseTrustedProxies(raw); err == nil {
			t.Errorf("accepted invalid trusted proxy %q", raw)
		}
	}
}

func TestBrowserPublicOrigin(t *testing.T) {
	if err := ValidateBrowserPublicOrigin("http://localhost:8080", "development"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBrowserPublicOrigin("http://example.com", "production"); err == nil {
		t.Fatal("production HTTP origin accepted")
	}
	if err := ValidateBrowserPublicOrigin("https://example.com/path", "production"); err == nil {
		t.Fatal("path-bearing origin accepted")
	}
}
