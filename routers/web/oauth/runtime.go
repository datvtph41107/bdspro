package oauth

import (
	"errors"
	"sync"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
)

var processBrowser struct {
	mu      sync.RWMutex
	browser *Browser
	started bool
}

// Init owns browser OAuth session transport setup. No router constructor
// initializes cookies or protocol configuration.
func Init() error {
	cfg := setting.WebConfig()
	var browser *Browser
	if cfg.BrowserPublicOrigin != "" {
		if err := setting.ValidateBrowserPublicOrigin(cfg.BrowserPublicOrigin, cfg.Environment); err != nil {
			return err
		}
		codec, err := authcrypto.NewBrowserSessionCodec(cfg.TokenSymmetricKey)
		if err != nil {
			return err
		}
		browser, err = NewBrowser(Config{
			Codec:        codec,
			PublicOrigin: cfg.BrowserPublicOrigin,
			OIDCEnabled:  cfg.OIDC.Enabled() && cfg.OIDC.RedirectURI == cfg.BrowserPublicOrigin+"/users/oauth2/callback",
		})
		if err != nil {
			return err
		}
	}
	processBrowser.mu.Lock()
	defer processBrowser.mu.Unlock()
	if processBrowser.started {
		return errors.New("browser oauth: already initialized")
	}
	processBrowser.browser, processBrowser.started = browser, true
	return nil
}

func ProcessBrowser() *Browser {
	processBrowser.mu.RLock()
	defer processBrowser.mu.RUnlock()
	if !processBrowser.started {
		panic("browser oauth: subsystem not initialized")
	}
	return processBrowser.browser
}
