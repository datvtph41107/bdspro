package auth

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func RequestPasswordReset(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.PasswordResetRequest](ctx)
	email, err := authservice.NormalizeEmail(form.Email)
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	if !setting.Mailer.Enabled() {
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "password recovery delivery unavailable")
		return
	}
	if !subjectAdmission(ctx, ipactive.RequestPasswordReset, email) {
		return
	}
	if err := authservice.RequestPasswordReset(ctx, email); err != nil {
		writeAuthError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, map[string]string{"status": "accepted"})
}

func ConfirmPasswordReset(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.PasswordResetConfirm](ctx)
	if err := authservice.ResetPassword(ctx, form.ResetToken, form.NewPassword); err != nil {
		writeAuthError(ctx, err)
		return
	}
	ClearRefreshCookie(ctx)
	ctx.Status(http.StatusNoContent)
}

func ChangePassword(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return
	}
	if !subjectAdmission(ctx, ipactive.Login, "account:"+strconv.FormatInt(ctx.Doer.AccountID, 10)) {
		return
	}
	form := web.GetForm[*structs.PasswordChangeRequest](ctx)
	result, err := authservice.ChangePassword(ctx, *ctx.Doer, form.CurrentPassword, form.NewPassword, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeAuthError(ctx, err)
		return
	}
	writeAuthenticated(ctx, result)
}
