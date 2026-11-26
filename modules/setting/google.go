package setting

import (
	"errors"
	"strings"
)

type GoogleSettings struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

var Google GoogleSettings

func (g GoogleSettings) Enabled() bool { return g.ClientID != "" }

func loadGoogleFrom(cfg ConfigProvider, server ServerSettings) (GoogleSettings, error) {
	sec := cfg.Section("google")
	g := GoogleSettings{
		ClientID:     sec.Key("CLIENT_ID").String(),
		ClientSecret: sec.Key("CLIENT_SECRET").String(),
		CallbackURL:  sec.Key("CALLBACK_URL").String(),
	}
	if g == (GoogleSettings{}) {
		return g, nil
	}
	if len(g.ClientID) > 512 || !strings.HasSuffix(g.ClientID, ".apps.googleusercontent.com") ||
		len(g.ClientSecret) < 16 || len(g.ClientSecret) > 512 ||
		strings.ContainsAny(g.ClientID+g.ClientSecret, " \r\n\t") ||
		server.BrowserPublicOrigin == "" || g.CallbackURL != server.BrowserPublicOrigin+"/api/v1/auth/google/callback" {
		return GoogleSettings{}, errors.New("invalid google client credentials or callback URL; use the configured public origin")
	}
	return g, nil
}
