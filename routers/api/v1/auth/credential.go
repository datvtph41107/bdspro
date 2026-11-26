package auth

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	delivery "github.com/datvtph41107/bdspro/modules/mail"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func credentialReady(ctx *appctx.APIContext, action ipactive.Action) bool {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return false
	}
	return subjectAdmission(ctx, action, "account:"+strconv.FormatInt(ctx.Doer.AccountID, 10))
}

func ReauthenticatePassword(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.Login) {
		return
	}
	form := web.GetForm[*structs.PasswordReauthRequest](ctx)
	result, err := authservice.ReauthenticatePassword(ctx, *ctx.Doer, form.Password, form.Action)
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func RequestPasswordSetProof(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.StartEmailLink) {
		return
	}
	err := authservice.RequestPasswordSetProof(ctx, *ctx.Doer)
	if errors.Is(err, delivery.ErrUnavailable) {
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "email delivery unavailable")
		return
	}
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, map[string]string{"status": "accepted"})
}

func SetPassword(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.Login) {
		return
	}
	form := web.GetForm[*structs.PasswordSetRequest](ctx)
	result, err := authservice.SetPassword(ctx, *ctx.Doer, form.ReauthenticationProof, form.NewPassword, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	writeAuthenticated(ctx, result)
}

func RemoveCredential(kind string) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if !credentialReady(ctx, ipactive.Login) {
			return
		}
		form := web.GetForm[*structs.RecentAuthRequest](ctx)
		result, err := authservice.RemoveCredential(ctx, *ctx.Doer, kind, form.ReauthenticationProof, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
		if err != nil {
			writeAuthError(ctx, err)
			return
		}
		writeAuthenticated(ctx, result)
	}
}

func RevokeAllSessions(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.RevokeToken) {
		return
	}
	form := web.GetForm[*structs.RecentAuthRequest](ctx)
	if err := authservice.RevokeAllSessions(ctx, *ctx.Doer, form.ReauthenticationProof); err != nil {
		writeAuthError(ctx, err)
		return
	}
	ClearRefreshCookie(ctx)
	ctx.Status(http.StatusNoContent)
}
