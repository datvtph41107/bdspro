package auth

import (
	"errors"
	"net/http"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// The initial browser binding is a transport lifetime, not a Google flow deadline.
const googleBrowserCookieMaxAge = 30 * 60

func googleCookieName() string {
	if secureCookie() {
		return "__Host-bdspro-google"
	}
	return "bdspro_google_dev"
}

func googleHeaders(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
}

func googleBrowser(ctx *appctx.APIContext) (string, []byte, error) {
	cookie, err := ctx.CurrentRequest().Cookie(googleCookieName())
	if err != nil {
		return "", nil, authservice.ErrRegistrationProof
	}
	digest, err := token.RecoveryDigest(cookie.Value)
	if err != nil {
		return "", nil, authservice.ErrRegistrationProof
	}
	return cookie.Value, digest, nil
}

func writeGoogleError(ctx *appctx.APIContext, err error) {
	switch {
	case errors.Is(err, token.ErrGoogleUnavailable):
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "Google authentication unavailable")
	case errors.Is(err, token.ErrGoogleProof):
		writeAuthError(ctx, authservice.ErrRegistrationProof)
	default:
		writeAuthError(ctx, err)
	}
}

func GoogleStart(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	remember := ctx.CurrentRequest().URL.Query().Get("stay_signed_in")
	if remember != "" && remember != "true" && remember != "false" {
		ctx.APIError(http.StatusBadRequest, "invalid_request", "invalid stay_signed_in value")
		return
	}
	result, err := authservice.StartGoogle(ctx, remember == "true")
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	setGoogleCookie(ctx, result.Browser, googleBrowserCookieMaxAge)
	ctx.Header("Location", result.URL)
	ctx.Status(http.StatusFound)
}

func GoogleCallback(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	if !setting.Google.Enabled() {
		writeGoogleError(ctx, token.ErrGoogleUnavailable)
		return
	}

	location := setting.Server.BrowserPublicOrigin + "/auth/google/return"
	_, browser, err := googleBrowser(ctx)
	if err == nil {
		query := ctx.CurrentRequest().URL.Query()
		state, code := query.Get("state"), query.Get("code")
		if len(query["state"]) != 1 || len(query["code"]) != 1 || query.Has("error") {
			code = ""
		}
		err = authservice.GoogleCallback(ctx, browser, state, code)
	}
	if err != nil {
		location += "?status=failed"
	}
	ctx.Header("Location", location)
	ctx.Status(http.StatusSeeOther)
}

func GoogleResult(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	rawBrowser, browser, err := googleBrowser(ctx)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	result, err := authservice.ConsumeGoogleResult(ctx, browser, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	if result.Login != nil {
		writeAuthenticated(ctx, *result.Login)
		return
	}
	if result.Proof.Status == "sign_in_required" {
		ctx.JSON(http.StatusOK, map[string]string{"status": result.Proof.Status})
		return
	}
	if result.Proof.Status == "registration_required" {
		// Retain the same browser binding for the newly issued proof's lifetime.
		setGoogleCookie(ctx, rawBrowser, result.Proof.ExpiresIn)
	}
	ctx.JSON(http.StatusOK, result.Proof)
}

func GoogleComplete(ctx *appctx.APIContext) {
	googleHeaders(ctx)
	form := web.GetForm[*structs.GoogleComplete](ctx)
	_, browser, err := googleBrowser(ctx)
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	result, err := authservice.CompleteGoogle(ctx, form.RegistrationToken, browser, form.Username, form.TermsVersion,
		form.MarketingOptIn, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeGoogleError(ctx, err)
		return
	}
	writeAuthenticatedStatus(ctx, result, http.StatusCreated)
}

func setGoogleCookie(ctx *appctx.APIContext, browser string, maxAge int) {
	ctx.Header("Set-Cookie", (&http.Cookie{
		Name: googleCookieName(), Value: browser, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: secureCookie(), SameSite: http.SameSiteLaxMode,
	}).String())
}
