package auth

import (
	"net/http"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
)

func (h *OTPHandler) StartEmailLink(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.StartEmailLink) {
		return
	}
	form := web.GetForm[*structs.EmailLinkStart](ctx)
	email, err := authservice.NormalizeEmail(form.Email)
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}

	if !subjectAdmission(ctx, ipactive.Register, email) {
		return
	}
	result, err := authservice.StartContactLink(ctx, h.delivery, *ctx.Doer, "email", email, form.ReauthenticationProof, "")
	if err != nil {
		writeRegistrationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func (h *OTPHandler) StartPhoneLink(ctx *appctx.APIContext) {
	if !credentialReady(ctx, ipactive.StartPhoneBind) {
		return
	}
	form := web.GetForm[*structs.PhoneLinkStart](ctx)
	phone, err := token.NormalizePhone(form.Phone, form.CountryCode)
	if err != nil {
		writePhoneError(ctx, authservice.ErrPhoneInvalid)
		return
	}
	if _, err := h.delivery.ResolveChannel(form.Channel); err != nil {
		writePhoneError(ctx, err)
		return
	}
	if !smsAdmission(ctx, phone) {
		return
	}
	result, err := authservice.StartContactLink(ctx, h.delivery, *ctx.Doer, "phone", phone, form.ReauthenticationProof, form.Channel)
	if err != nil {
		writePhoneError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func VerifyContactLink(kind string, action ipactive.Action) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if !credentialReady(ctx, action) {
			return
		}
		form := web.GetForm[*structs.ContactLinkVerify](ctx)
		id, err := uuid.Parse(form.ChallengeID)
		if err != nil || id == uuid.Nil || id.String() != form.ChallengeID {
			writeRegistrationError(ctx, authservice.ErrRegistrationCode)
			return
		}
		destination, err := authservice.ContactLinkDestination(ctx, *ctx.Doer, kind, id)
		if err != nil {
			writeRegistrationError(ctx, err)
			return
		}
		limit := ipactive.VerifyEmail
		if kind == "phone" {
			limit = ipactive.SMSVerifyDestination
		}
		if !subjectAdmission(ctx, limit, destination) {
			return
		}
		result, err := authservice.VerifyContactLink(ctx, *ctx.Doer, kind, id, form.Code, ctx.ClientIP(), ctx.CurrentRequest().UserAgent())
		if err != nil {
			writeRegistrationError(ctx, err)
			return
		}
		writeAuthenticated(ctx, result)
	}
}
