package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBrowserCookieAttributes(t *testing.T) {
	c := BrowserSessionCookie("signed", true)
	if !c.Secure || !c.HttpOnly || c.Path != "/" || c.SameSite != http.SameSiteLaxMode || c.Name != "__Host-bdspro-browser" || c.Domain != "" {
		t.Fatalf("insecure cookie %+v", c)
	}
	d := BrowserSessionCookie("dev", false)
	if d.Name == c.Name || d.Secure {
		t.Fatalf("insecure production cookie %+v", d)
	}
}
func TestVerifyBrowserPostOrigin(t *testing.T) {
	good := httptest.NewRequest(http.MethodPost, "https://bdspro.example/oauth/authorize", nil)
	good.Header.Set("Origin", "https://bdspro.example")
	good.Header.Set("Sec-Fetch-Site", "same-origin")
	if err := VerifyBrowserPostOrigin(good, "https://bdspro.example"); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"", "null", "https://evil.example", "http://bdspro.example", "https://bdspro.example.evil.test", "https://bdspro.example:8443", "https://bdspro.example/any", "https://u@bdspro.example"} {
		r := httptest.NewRequest(http.MethodPost, "https://bdspro.example/oauth/authorize", nil)
		r.Header.Set("Origin", origin)
		if err := VerifyBrowserPostOrigin(r, "https://bdspro.example"); err == nil {
			t.Errorf("accepted origin %q", origin)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "https://bdspro.example/oauth/authorize", nil)
	r.Header.Set("Origin", "https://bdspro.example")
	r.Header.Set("Sec-Fetch-Site", "same-site")
	if err := VerifyBrowserPostOrigin(r, "https://bdspro.example"); err == nil {
		t.Fatal("cross-subdomain request accepted")
	}
}
func TestBrowserNextPath(t *testing.T) {
	for _, p := range []string{"/oauth/authorize", "/oauth/authorize?client_id=42&state=abc"} {
		got, err := BrowserNextPath(p)
		if err != nil || got != p {
			t.Fatalf("%q => %q %v", p, got, err)
		}
	}
	for _, p := range []string{"", "//evil.test/oauth/authorize", "https://evil.test/oauth/authorize", "/oauth/authorize", "/oauth/browser/%61uthorize", "/oauth/authorize#fragment", "/oauth/authorize?x=%zz", "/oauth/authorize\nLocation: https://evil.test"} {
		if _, err := BrowserNextPath(p); err == nil {
			t.Errorf("accepted return path %q", p)
		}
	}
}
