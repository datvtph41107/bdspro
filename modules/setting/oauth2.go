package setting

import (
    "fmt"
    "net/url"
    "strings"
)

// OAuth2Provider configures one trusted OIDC provider for account login.
type OAuth2Provider struct {
    Issuer string
    AuthorizationURL string
    TokenURL string
    JWKSURL string
    ClientID string
    ClientSecret string
    RedirectURI string
}

var OAuth2 OAuth2Provider

func loadOAuth2From(rootCfg ConfigProvider) error {
    sec := rootCfg.Section("oauth2")
    cfg := OAuth2Provider{
        Issuer: sec.Key("ISSUER").String(),
        AuthorizationURL: sec.Key("AUTHORIZATION_URL").String(),
        TokenURL: sec.Key("TOKEN_URL").String(),
        JWKSURL: sec.Key("JWKS_URL").String(),
        ClientID: sec.Key("CLIENT_ID").String(),
        ClientSecret: sec.Key("CLIENT_SECRET").String(),
        RedirectURI: sec.Key("REDIRECT_URI").String(),
    }
    legacy := rootCfg.Section("oauth")
    if legacy.Key("GOOGLE_CLIENT_ID").String() != "" ||
        legacy.Key("GOOGLE_CLIENT_SECRET").String() != "" ||
        legacy.Key("GOOGLE_REDIRECT_URI").String() != "" {
        return fmt.Errorf("legacy OAuth provider settings are unsupported; configure [oauth2]")
    }
    if cfg.Enabled() {
        if err := cfg.Validate(); err != nil { return err }
    }
    OAuth2 = cfg
    return nil
}

func (p OAuth2Provider) Enabled() bool {
    return p.Issuer != "" || p.AuthorizationURL != "" || p.TokenURL != "" ||
        p.JWKSURL != "" || p.ClientID != "" || p.ClientSecret != "" || p.RedirectURI != ""
}

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
    if p.ClientID == "" || p.ClientSecret == "" { return fmt.Errorf("oauth2 client credentials are required") }
    for _, item := range []struct {
        name string
        value string
        local bool
    }{
        {"issuer", p.Issuer, false},
        {"authorization URL", p.AuthorizationURL, false},
        {"token URL", p.TokenURL, false},
        {"JWKS URL", p.JWKSURL, false},
        {"redirect URI", p.RedirectURI, true},
    } {
        if err := check(item.name, item.value, item.local); err != nil { return err }
    }
    callback, err := url.Parse(p.RedirectURI)
    if err != nil || callback.Path != "/users/oauth2/callback" {
        return fmt.Errorf("oauth2 redirect URI must target /users/oauth2/callback")
    }
    return nil
}
