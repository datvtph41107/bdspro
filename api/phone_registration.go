package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

// Phone-only registration is a distinct anonymous intent from phone login.
// Starting a registration does not create an Account or grant a session.
func (server *Server) registerPhoneStart(ctx *gin.Context) {
	var req phoneLoginStartRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if ok, retry := server.authLimiter.allow("phone-register-identity", fingerprint(phone),
		ratePolicy{burst: 6, window: 15 * time.Minute}, time.Now()); !ok {
		authRateLimitExceeded(ctx, retry)
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
		return
	}
	result, err := server.authentication.StartPhoneRegistration(ctx.Request.Context(), phone)
	switch {
	case errors.Is(err, auth.ErrPhoneInvalid):
		writeAPIError(ctx, http.StatusBadRequest, "invalid phone")
		return
	case errors.Is(err, auth.ErrSMSUnavailable):
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
		return
	case errors.Is(err, auth.ErrPhoneRegistrationDelivery):
		server.logger.WarnContext(ctx.Request.Context(), "phone_registration_delivery_failed",
			"request_id", ctx.GetString("request_id"))
	case err != nil:
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
		return
	}
	replyJSON(ctx, http.StatusAccepted, gin.H{
		"challenge_token": result.ChallengeToken, "expires_in_seconds": 300,
	})
}

func (server *Server) registerPhoneVerify(ctx *gin.Context) {
	var req phoneLoginVerifyRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	if ok, retry := server.authLimiter.allow("phone-register-challenge", fingerprint(req.ChallengeToken),
		ratePolicy{burst: 10, window: 15 * time.Minute}, time.Now()); !ok {
		authRateLimitExceeded(ctx, retry)
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
		return
	}
	result, err := server.authentication.VerifyPhoneRegistration(ctx.Request.Context(), req.ChallengeToken, req.Code)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAPIError(ctx, http.StatusUnauthorized, "invalid or expired code")
	case errors.Is(err, auth.ErrPhoneInUse):
		// The caller has proved possession of a valid registration code,
		// but the phone is already bound. Do not log in to another Account.
		writeAPIError(ctx, http.StatusConflict, "phone unavailable; use login")
	case errors.Is(err, auth.ErrSMSUnavailable):
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
	case err != nil:
		writeAPIError(ctx, http.StatusServiceUnavailable, "registration unavailable")
	default:
		server.respondLoginSession(ctx, result)
	}
}
