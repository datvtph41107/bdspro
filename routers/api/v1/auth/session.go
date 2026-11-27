package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

const refreshCookieName = "bdspro_refresh"

func BrowserMutation(ctx *appctx.APIContext) {
	r := ctx.CurrentRequest()
	if r.Header.Get("X-BDS-Request") != "1" || token.VerifyBrowserPostOrigin(r, setting.Server.BrowserPublicOrigin) != nil {
		ctx.Data[appctx.ErrorReasonKey] = "browser_origin"
		ctx.APIError(http.StatusForbidden, "permission_denied", "browser origin verification failed")
	}
}

func Login(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.LoginRequest](ctx)
	if !subjectAdmission(ctx, ipactive.Login, strings.ToLower(strings.TrimSpace(form.Identifier))) {
		return
	}
	result, err := authservice.LoginPassword(ctx, form.Identifier, form.Password, ctx.ClientIP(), ctx.CurrentRequest().UserAgent(), form.StaySignedIn)
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	writeAuthenticated(ctx, result)
}

func subjectAdmission(ctx *appctx.APIContext, action ipactive.Action, subject string) bool {
	limiter, ok := ipactive.LookupMiddleware(ctx)
	if !ok {
		return ctx.Admit(action, "subject", ipactive.Event{}, ipactive.ErrUnavailable)
	}
	mac := hmac.New(sha256.New, []byte(setting.Security.TokenSymmetricKey))
	_, _ = mac.Write([]byte("bdspro.auth.subject.v1\x00" + string(action) + "\x00" + subject))
	event, err := limiter.CheckSubject(ctx, action, hex.EncodeToString(mac.Sum(nil)))
	return ctx.Admit(action, "subject", event, err)
}

func Refresh(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	cookie, err := ctx.CurrentRequest().Cookie(refreshCookieName)
	if err != nil {
		ctx.AuthenticationRequired()
		return
	}
	result, err := authservice.RefreshSession(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, util.ErrUnauthenticated) {
			ClearRefreshCookie(ctx)
		}
		writeAuthError(ctx, err)
		return
	}
	writeAuthenticated(ctx, result)
}

func Logout(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	r := ctx.CurrentRequest()
	if r.Header.Get("Authorization") != "" {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.AuthenticationRequired()
			return
		}
		actor, err := authservice.ResolveAccessToken(ctx, parts[1])
		if err != nil {
			writeAuthError(ctx, err)
			return
		}
		if err := authservice.RevokeSession(ctx, *actor, actor.SessionID); err != nil {
			writeAuthError(ctx, err)
			return
		}
	} else if cookie, err := r.Cookie(refreshCookieName); err == nil {
		if err := authservice.LogoutRefresh(ctx, cookie.Value); err != nil && !errors.Is(err, util.ErrUnauthenticated) {
			writeAuthError(ctx, err)
			return
		}
	}
	ClearRefreshCookie(ctx)
	ctx.Status(http.StatusNoContent)
}

func secureCookie() bool {
	return setting.Common.RunMode != "development" || strings.HasPrefix(setting.Server.BrowserPublicOrigin, "https://")
}

func writeAuthenticated(ctx *appctx.APIContext, result authservice.LoginResult) {
	writeAuthenticatedStatus(ctx, result, http.StatusOK)
}

func writeAuthenticatedStatus(ctx *appctx.APIContext, result authservice.LoginResult, status int) {
	cookie := &http.Cookie{
		Name: refreshCookieName, Value: result.Refresh, Path: "/api/v1",
		HttpOnly: true, Secure: secureCookie(), SameSite: http.SameSiteLaxMode,
	}
	if result.Remember {
		cookie.Expires = result.ExpiresAt
		cookie.MaxAge = max(1, int(time.Until(result.ExpiresAt)/time.Second))
	}
	ctx.Header("Set-Cookie", cookie.String())
	ctx.JSON(status, result)
}

func ClearRefreshCookie(ctx *appctx.APIContext) {
	cookie := &http.Cookie{
		Name: refreshCookieName, Value: "", Path: "/api/v1", MaxAge: -1,
		Expires: time.Unix(1, 0).UTC(), HttpOnly: true, Secure: secureCookie(), SameSite: http.SameSiteLaxMode,
	}
	ctx.Header("Set-Cookie", cookie.String())
}

func writeAuthError(ctx *appctx.APIContext, err error) {
	switch {
	case errors.Is(err, authservice.ErrInvalidCredentials):
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.APIError(http.StatusUnauthorized, "invalid_credentials", "invalid email, username or password")
	case errors.Is(err, util.ErrUnauthenticated):
		ctx.AuthenticationRequired()
	case errors.Is(err, token.ErrPasswordBusy):
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "authentication capacity unavailable")
	default:
		ctx.APIErrorAuto(err)
	}
}
