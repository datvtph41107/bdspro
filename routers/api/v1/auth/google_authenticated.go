package auth

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func GoogleLinkStart(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	if !credentialReady(ctx, ipactive.StartOAuth2Link) {
		return
	}
	form := web.GetForm[*structs.RecentAuthRequest](ctx)
	startAuthenticatedGoogle(ctx, "link", "google.link", form.ReauthenticationProof)
}

func GoogleReauthStart(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	if !credentialReady(ctx, ipactive.StartOAuth2Link) {
		return
	}
	form := web.GetForm[*structs.GoogleReauthRequest](ctx)
	startAuthenticatedGoogle(ctx, "reauth", form.Action, "")
}

func startAuthenticatedGoogle(ctx *appctx.APIContext, purpose, action, proof string) {
	result, err := authservice.StartAuthenticatedGoogle(ctx, *ctx.Doer, purpose, action, proof)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	setGoogleCookie(ctx, result.Browser, googleBrowserCookieMaxAge)
	ctx.JSON(http.StatusOK, map[string]string{"authorization_url": result.URL})
}

func GoogleLinkResult(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	if !credentialReady(ctx, ipactive.StartOAuth2Link) {
		return
	}
	_, browser, err := googleBrowser(ctx)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	result, err := authservice.CompleteGoogleLink(ctx, *ctx.Doer, browser, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	writeAuthenticated(ctx, result)
}

func GoogleReauthResult(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	if !credentialReady(ctx, ipactive.StartOAuth2Link) {
		return
	}
	_, browser, err := googleBrowser(ctx)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	result, err := authservice.CompleteGoogleReauth(ctx, *ctx.Doer, browser)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}
