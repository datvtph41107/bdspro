package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// BrowserCookieName uses the prefix enforced by modern browsers in production.
// The development variant MUST only be used for loopback HTTP development.
func BrowserCookieName(secure bool) string {
	if secure {
		return "__Host-bdspro-browser"
	}
	return "bdspro_browser_dev"
}

func BrowserSessionCookie(value string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name: BrowserCookieName(secure), Value: value, Path: "/",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		MaxAge: int(BrowserSessionTTL.Seconds()),
	}
}

// VerifyBrowserPostOrigin requires a configured, trusted public origin.
// Never derive expectedOrigin from Host or X-Forwarded-Host request headers.
// Must run on every browser state-changing POST in addition to a session-
// bound CSRF token. A reverse proxy must validate allowed Host separately.
func VerifyBrowserPostOrigin(r *http.Request, expectedOrigin string) error {
	if r == nil || expectedOrigin == "" {
		return ErrBrowserProofInvalid
	}
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" || len(origin) > 512 {
		return ErrBrowserProofInvalid
	}
	source, err := url.Parse(origin)
	if err != nil || source == nil || source.Opaque != "" || source.User != nil || source.RawQuery != "" || source.Fragment != "" || source.Path != "" || source.Host == "" {
		return ErrBrowserProofInvalid
	}
	if source.Scheme != "https" && source.Scheme != "http" {
		return ErrBrowserProofInvalid
	}
	configured, err := url.Parse(expectedOrigin)
	if err != nil || configured == nil || configured.Scheme != "https" && configured.Scheme != "http" ||
		configured.Host == "" || configured.Path != "" || configured.RawQuery != "" || configured.Fragment != "" || configured.User != nil {
		return errors.New("invalid configured browser public origin")
	}
	// Requiring exact origin protects port and scheme as well as host.
	if source.String() != configured.String() {
		return ErrBrowserProofInvalid
	}
	// Strict enough to reject cross-subdomain requests even when SameSite=Lax.
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return ErrBrowserProofInvalid
	}
	return nil
}

// BrowserNextPath is intentionally restricted to the OAuth consent page,
// never an arbitrary return URL or a raw client-provided redirect_uri.
func BrowserNextPath(raw string) (string, error) {
	if len(raw) == 0 || len(raw) > 4096 || strings.ContainsAny(raw, "\r\n\\") {
		return "", ErrBrowserProofInvalid
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u == nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || u.EscapedPath() != "/oauth/authorize" {
		return "", ErrBrowserProofInvalid
	}
	if _, err := url.ParseQuery(u.RawQuery); err != nil {
		return "", ErrBrowserProofInvalid
	}
	return u.RequestURI(), nil
}
