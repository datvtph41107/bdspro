package user

import (
	"errors"
	"net/http"
	"unicode/utf8"
	"strconv"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/datvtph41107/bdspro/templates"
)

// Register is a typed Gitea-style HTTP action. Credential creation stays in
// AuthService, while the security limiter is shared with legacy endpoints.
func Register(ctx *appctx.APIContext) {
	var req structs.RegisterRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "register", req.Email, 10, 3, time.Hour) {
		return
	}
	account, err := auth_service.FromContext(ctx).Register(ctx, req.Email, req.Password)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToUser(account))
}

// Login is a typed API action. It does not inspect Gin keys, trust raw proxy
// headers, or create a second session-issuance implementation.
func Login(ctx *appctx.APIContext) {
	var req structs.LoginRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "login", req.Email, 30, 10, time.Minute) {
		return
	}
	result, err := auth_service.FromContext(ctx).Login(ctx, req.Email, req.Password,
		ctx.CurrentRequest().UserAgent(), ctx.ClientIP())
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToLogin(result))
}

func StartPhoneBind(ctx *appctx.APIContext) {
	var req structs.PhoneBindRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_bind_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).StartPhoneBind(ctx, actor.AccountID, req.Password, req.Phone); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}

func VerifyPhoneBind(ctx *appctx.APIContext) {
	var req structs.PhoneVerifyRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_bind_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	if err := auth_service.FromContext(ctx).VerifyPhoneBind(ctx, actor.AccountID, req.Phone, req.Code); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Phone signup and login are public; no AccountID is available until OTP
// succeeds. The same IP/credential admission contract applies to each action.
func StartPhoneRegister(ctx *appctx.APIContext) {
	var req structs.PhoneStartRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_register_start", req.Phone, 10, 3, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).StartPhoneRegister(ctx, req.Phone); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}
func VerifyPhoneRegister(ctx *appctx.APIContext) {
	var req structs.PhoneVerifyRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_register_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := auth_service.FromContext(ctx).VerifyPhoneRegister(ctx, req.Phone, req.Code, ctx.CurrentRequest().UserAgent(), ctx.ClientIP())
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToLogin(result))
}
func StartPhoneLogin(ctx *appctx.APIContext) {
	var req structs.PhoneStartRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_login_start", req.Phone, 30, 10, 10*time.Minute) {
		return
	}
	if err := auth_service.FromContext(ctx).StartPhoneLogin(ctx, req.Phone); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}
func VerifyPhoneLogin(ctx *appctx.APIContext) {
	var req structs.PhoneVerifyRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "phone_login_verify", req.Phone, 40, 5, 10*time.Minute) {
		return
	}
	result, err := auth_service.FromContext(ctx).VerifyPhoneLogin(ctx, req.Phone, req.Code, ctx.CurrentRequest().UserAgent(), ctx.ClientIP())
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToLogin(result))
}

func StartEmailLink(ctx *appctx.APIContext) {
	var req structs.EmailLinkStartRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "email_link_start", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).StartEmailLink(ctx, actor.AccountID, req.Email, req.PhoneCode); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}

func ConfirmEmailLink(ctx *appctx.APIContext) {
	var req structs.EmailLinkConfirmRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "email_link_verify", req.Token, 20, 5, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).ConfirmEmailLink(ctx, actor.AccountID, req.Token); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// RequestPasswordReset owns only HTTP input/admission; non-enumerating reset
// behavior remains in AuthService.
func RequestPasswordReset(ctx *appctx.APIContext) {
	var req structs.PasswordResetRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "password_reset", req.Email, 10, 3, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).RequestPasswordReset(ctx, req.Email); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}

// GET is a non-consuming preview; passwords are submitted only with POST.
func PasswordResetPage(ctx *appctx.APIContext, w http.ResponseWriter) {
	token := ctx.CurrentRequest().URL.Query().Get("token")
	if len(token) != 43 {
		ctx.APIError(http.StatusBadRequest, "invalid_reset_token", "invalid or expired reset token")
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	_ = templates.PasswordResetPage.Execute(w, token)
}

func ResetPassword(ctx *appctx.APIContext) {
	var req structs.PasswordResetConfirmRequest
	var err error
	if strings.HasPrefix(ctx.CurrentRequest().Header.Get("Content-Type"), "application/json") {
		err = ctx.BindJSON(&req)
	} else {
		var values = ctx.CurrentRequest()
		if err = values.ParseForm(); err == nil {
			req.Token = values.FormValue("token")
			req.Password = values.FormValue("password")
			length := utf8.RuneCountInString(req.Password)
			if req.Token == "" || length < 8 || length > 72 {
				err = errors.New("invalid reset form")
			}
		}
	}
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "password_reset_confirm", req.Token, 20, 6, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).ResetPassword(ctx, req.Token, req.Password); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func ResendVerificationEmail(ctx *appctx.APIContext) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "verify_email_resend", req.Email, 10, 2, time.Hour) {
		return
	}
	if err := auth_service.FromContext(ctx).ResendVerificationEmail(ctx, req.Email); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusAccepted)
}

// VerifyEmailPage is safe for mail scanners: GET never consumes a token.
func VerifyEmailPage(ctx *appctx.APIContext, w http.ResponseWriter) {
	values := ctx.CurrentRequest().URL.Query()
	id, err := strconv.ParseInt(values.Get("email_id"), 10, 64)
	if err != nil || id <= 0 || values.Get("secret_code") == "" {
		ctx.APIError(http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	req := structs.VerifyEmailQuery{EmailID: id, SecretCode: values.Get("secret_code")}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	_ = templates.VerifyEmailPage.Execute(w, req)
}

func VerifyEmail(ctx *appctx.APIContext) {
	if err := ctx.CurrentRequest().ParseForm(); err != nil {
		ctx.APIError(http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	values := ctx.CurrentRequest().Form
	id, err := strconv.ParseInt(values.Get("email_id"), 10, 64)
	if err != nil || id <= 0 || values.Get("secret_code") == "" {
		ctx.APIError(http.StatusBadRequest, "invalid_verification_link", "invalid verification link")
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "verify_email", strconv.FormatInt(id, 10), 60, 10, time.Minute) {
		return
	}
	if err := auth_service.FromContext(ctx).VerifyEmail(ctx, id, values.Get("secret_code")); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"verified": true})
}

// Renew uses the same refresh-session service and Redis admission as before,
// but the HTTP action now consumes only APIContext, not Gin request state.
func Renew(ctx *appctx.APIContext) {
	var req structs.TokenRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "refresh", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	result, err := auth_service.FromContext(ctx).Renew(ctx, req.RefreshToken)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, structs.RenewAccessTokenResponse{
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt,
	})
}

// Revoke preserves the existing refresh-token ownership and revocation
// semantics; request scope and response emission belong to APIContext.
func Revoke(ctx *appctx.APIContext) {
	var req structs.TokenRequest
	if err := ctx.BindJSON(&req); err != nil {
		ctx.APIInvalid(err)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "revoke", req.RefreshToken, 60, 20, time.Minute) {
		return
	}
	if err := auth_service.FromContext(ctx).Revoke(ctx, req.RefreshToken); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// GetConnections returns a bounded history of this Account's session issuance.
// The account identifier is always taken from the authenticated request context.
func GetConnections(ctx *appctx.APIContext) {
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	connections, err := auth_service.FromContext(ctx).GetConnections(ctx, actor.AccountID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"connections": convert.ToConnections(connections)})
}
