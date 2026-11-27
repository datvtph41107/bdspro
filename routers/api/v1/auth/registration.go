package auth

import (
	"errors"
	"net/http"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/google/uuid"
)

func registrationCookieName() string {
	if secureCookie() {
		return "__Host-bdspro-register"
	}
	return "bdspro_register_dev"
}

func registrationBrowser(ctx *appctx.APIContext) ([]byte, error) {
	cookie, err := ctx.CurrentRequest().Cookie(registrationCookieName())
	if err != nil {
		return nil, authservice.ErrRegistrationProof
	}
	digest, err := token.RecoveryDigest(cookie.Value)
	if err != nil {
		return nil, authservice.ErrRegistrationProof
	}
	return digest, nil
}

func writeRegistrationError(ctx *appctx.APIContext, err error) {
	switch {
	case errors.Is(err, mailer.ErrUnavailable):
		if !setting.Mailer.Enabled() {
			ctx.Data[appctx.ErrorReasonKey] = "mailer_not_configured"
		} else {
			ctx.Data[appctx.ErrorReasonKey] = "email_delivery_failed"
		}
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "email delivery unavailable")
	case errors.Is(err, authservice.ErrRegistrationWait):
		ctx.Header("Retry-After", "60")
		ctx.APIErrorAuto(err)
	default:
		writeAuthError(ctx, err)
	}
}

func RegisterStart(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.RegistrationStart](ctx)
	email, err := authservice.NormalizeEmail(form.Email)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	if !setting.Mailer.Enabled() {
		writeRegistrationError(ctx, mailer.ErrUnavailable)
		return
	}
	if !subjectAdmission(ctx, ipactive.Register, email) {
		return
	}
	digest, err := ensureRegistrationBrowser(ctx)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	result, err := authservice.StartRegistration(ctx, email, digest)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func registrationFlow(ctx *appctx.APIContext, rawID string, action ipactive.Action) (uuid.UUID, []byte, bool) {
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil || id.String() != rawID {
		writeRegistrationError(ctx, authservice.ErrRegistrationProof)
		return uuid.Nil, nil, false
	}
	browser, err := registrationBrowser(ctx)
	if err != nil {
		writeRegistrationError(ctx, err)
		return uuid.Nil, nil, false
	}
	email, err := authservice.RegistrationDestination(ctx, id, browser)
	if err != nil {
		writeRegistrationError(ctx, err)
		return uuid.Nil, nil, false
	}
	if !subjectAdmission(ctx, action, email) {
		return uuid.Nil, nil, false
	}
	return id, browser, true
}

func RegisterResend(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if !setting.Mailer.Enabled() {
		writeRegistrationError(ctx, mailer.ErrUnavailable)
		return
	}
	form := web.GetForm[*structs.RegistrationResend](ctx)
	id, browser, ok := registrationFlow(ctx, form.ChallengeID, ipactive.Register)
	if !ok {
		return
	}
	result, err := authservice.ResendRegistration(ctx, id, browser)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func RegisterVerify(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.RegistrationVerify](ctx)
	id, browser, ok := registrationFlow(ctx, form.ChallengeID, ipactive.VerifyEmail)
	if !ok {
		return
	}
	result, err := authservice.VerifyRegistration(ctx, id, browser, form.Code)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	if result.Status == "sign_in_required" {
		ctx.JSON(http.StatusOK, map[string]string{"status": result.Status})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func RegisterComplete(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.RegistrationComplete](ctx)
	browser, err := registrationBrowser(ctx)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	result, err := authservice.CompleteRegistration(ctx, form.RegistrationToken, browser, form.Username,
		form.Password, form.TermsVersion, form.MarketingOptIn, form.StaySignedIn,
		ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	writeAuthenticatedStatus(ctx, result, http.StatusCreated)
}

func ensureRegistrationBrowser(ctx *appctx.APIContext) ([]byte, error) {
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return nil, err
	}
	if cookie, err := ctx.CurrentRequest().Cookie(registrationCookieName()); err == nil {
		if existing, err := token.RecoveryDigest(cookie.Value); err == nil {
			raw, digest = cookie.Value, existing
		}
	}

	ctx.Header("Set-Cookie", (&http.Cookie{
		Name: registrationCookieName(), Value: raw, Path: "/", MaxAge: 2700,
		HttpOnly: true, Secure: secureCookie(), SameSite: http.SameSiteLaxMode,
	}).String())
	return digest, nil
}
