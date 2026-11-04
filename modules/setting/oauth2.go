package setting

import (
	"fmt"
	"net/url"
	"strings"
)

// OAuth2Provider configures one trusted OpenID Connect provider.
// Authentication uses OAuth2 Authorization Code + PKCE and OIDC ID tokens.
// Providers are identified by their issuer, never by email or display name.
type OAuth2Provider struct {
	Issuer           string
	AuthorizationURL string
	TokenURL         string
	JWKSURL          string
	ClientID         string
	ClientSecret     string
	RedirectURI      string
}

// Enabled is true only if an issuer has been explicitly configured.
func (p OAuth2Provider) Enabled() bool { return p.Issuer != "" }

// Validate rejects incomplete or untrusted remote endpoint configuration.
// Redirects may use localhost HTTP only for local development.
func (p OAuth2Provider) Validate() error {
	check := func(field, value string, localhost bool) error {
		u, err := url.Parse(value)
		if err != nil || u == nil || u.Host == "" || u.User != nil ||
			u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
			return fmt.Errorf("oauth2 %s is not a valid absolute URL", field)
		}
		local := strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1"
		if u.Scheme != "https" && !(localhost && u.Scheme == "http" && local) {
			return fmt.Errorf("oauth2 %s must use HTTPS", field)
		}
		return nil
	}
	if p.ClientID == "" || p.ClientSecret == "" {
		return fmt.Errorf("oauth2 client credentials are required")
	}
	for _, item := range []struct {
		name, value string
		local       bool
	}{
		{"issuer", p.Issuer, false},
		{"authorization URL", p.AuthorizationURL, false},
		{"token URL", p.TokenURL, false},
		{"JWKS URL", p.JWKSURL, false},
		{"redirect URI", p.RedirectURI, true},
	} {
		if err := check(item.name, item.value, item.local); err != nil {
			return err
		}
	}
	return nil
}
