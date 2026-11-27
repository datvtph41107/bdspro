package auth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/datvtph41107/bdspro/modules/setting"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// BrowserMutation guards browser-sensitive mutations before the action executes.
func BrowserMutation(ctx *appctx.APIContext) {
	r := ctx.CurrentRequest()
	if r.Header.Get("X-BDS-Request") != "1" || !validBrowserOrigin(r, setting.Server.BrowserPublicOrigin) {
		ctx.Data[appctx.ErrorReasonKey] = "browser_origin"
		ctx.APIError(http.StatusForbidden, "permission_denied", "browser origin verification failed")
	}
}

// The request origin and fetch metadata are independent of bearer/session authority.
func validBrowserOrigin(r *http.Request, expectedOrigin string) bool {
	if r == nil || expectedOrigin == "" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" || len(origin) > 512 {
		return false
	}
	source, err := url.Parse(origin)
	if err != nil || source == nil || source.Opaque != "" || source.User != nil || source.RawQuery != "" || source.Fragment != "" || source.Path != "" || source.Host == "" {
		return false
	}
	if source.Scheme != "https" && source.Scheme != "http" {
		return false
	}
	configured, err := url.Parse(expectedOrigin)
	if err != nil || configured == nil || configured.Scheme != "https" && configured.Scheme != "http" ||
		configured.Host == "" || configured.Path != "" || configured.RawQuery != "" || configured.Fragment != "" || configured.User != nil {
		return false
	}
	if source.String() != configured.String() {
		return false
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	return true
}

func secureCookie() bool {
	return setting.Common.RunMode != "development" || strings.HasPrefix(setting.Server.BrowserPublicOrigin, "https://")
}
