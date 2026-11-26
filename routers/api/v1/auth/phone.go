package auth

import (
	"errors"
	"net/http"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/sms"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
)

func writePhoneError(ctx *appctx.APIContext, err error) {
	if errors.Is(err, sms.ErrUnavailable) {
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "SMS delivery unavailable")
		return
	}
	writeRegistrationError(ctx, err)
}

func smsAdmission(ctx *appctx.APIContext, phone string) bool {
	return subjectAdmission(ctx, ipactive.SMSDestinationCooldown, phone) &&
		subjectAdmission(ctx, ipactive.SMSDestinationHour, phone) &&
		subjectAdmission(ctx, ipactive.SMSDestinationDay, phone) &&
		subjectAdmission(ctx, ipactive.SMSGlobalDay, "all")
}

func PhoneStart(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.PhoneStart](ctx)
	phone, err := token.NormalizePhone(form.Phone, form.CountryCode)
	if err != nil {
		writePhoneError(ctx, authservice.ErrPhoneInvalid)
		return
	}
	if !setting.SMS.Enabled() {
		writePhoneError(ctx, sms.ErrUnavailable)
		return
	}
	if !smsAdmission(ctx, phone) {
		return
	}
	browser, err := ensureRegistrationBrowser(ctx)
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	result, err := authservice.StartPhone(ctx, phone, browser)
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func phoneFlow(ctx *appctx.APIContext, rawID string) (uuid.UUID, []byte, string, bool) {
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil || id.String() != rawID {
		writePhoneError(ctx, authservice.ErrRegistrationProof)
		return uuid.Nil, nil, "", false
	}
	browser, err := registrationBrowser(ctx)
	if err != nil {
		writePhoneError(ctx, err)
		return uuid.Nil, nil, "", false
	}
	phone, err := authservice.PhoneDestination(ctx, id, browser)
	if err != nil {
		writePhoneError(ctx, err)
		return uuid.Nil, nil, "", false
	}
	return id, browser, phone, true
}

func PhoneResend(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if !setting.SMS.Enabled() {
		writePhoneError(ctx, sms.ErrUnavailable)
		return
	}
	form := web.GetForm[*structs.RegistrationResend](ctx)
	id, browser, phone, ok := phoneFlow(ctx, form.ChallengeID)
	if !ok || !smsAdmission(ctx, phone) {
		return
	}
	result, err := authservice.ResendPhone(ctx, id, browser)
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func PhoneVerify(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.PhoneVerify](ctx)
	id, browser, phone, ok := phoneFlow(ctx, form.ChallengeID)
	if !ok || !subjectAdmission(ctx, ipactive.SMSVerifyDestination, phone) {
		return
	}
	result, err := authservice.VerifyPhone(ctx, id, browser, form.Code, form.StaySignedIn, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	if result.Login != nil {
		writeAuthenticated(ctx, *result.Login)
		return
	}
	ctx.JSON(http.StatusOK, result.Proof)
}

func PhoneComplete(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.PhoneComplete](ctx)
	browser, err := registrationBrowser(ctx)
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	result, err := authservice.CompletePhone(ctx, form.RegistrationToken, browser, form.Username, form.TermsVersion,
		form.MarketingOptIn, form.StaySignedIn, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	writeAuthenticatedStatus(ctx, result, http.StatusCreated)
}
