package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

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

	if source.String() != configured.String() {
		return ErrBrowserProofInvalid
	}

	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return ErrBrowserProofInvalid
	}
	return nil
}

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
