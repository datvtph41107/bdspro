package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
)

type phoneLoginStartRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type phoneLoginVerifyRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required"`
	Code           string `json:"code" binding:"required"`
}

// Anonymous phone login does not create or expose an Account until a valid
// SMS code is redeemed. Unknown and throttled phones receive a fake handle.
func (server *Server) loginPhoneStart(ctx *gin.Context) {
	var req phoneLoginStartRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	phone := strings.TrimSpace(req.Phone)
	// The per-identifier limiter applies equally to known and unknown phones.
	if ok, retry := server.authLimiter.allow("phone-login-identity", fingerprint(phone),
		ratePolicy{burst: 6, window: 15 * time.Minute}, time.Now()); !ok {
		authRateLimitExceeded(ctx, retry)
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
		return
	}
	result, err := server.authentication.StartPhoneLogin(ctx.Request.Context(), phone)
	switch {
	case errors.Is(err, auth.ErrPhoneInvalid):
		writeAPIError(ctx, http.StatusBadRequest, "invalid phone")
		return
	case errors.Is(err, auth.ErrSMSUnavailable):
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
		return
	case errors.Is(err, auth.ErrPhoneLoginDelivery):
		// Never reveal an existing Account by returning a different status.
		server.logger.WarnContext(ctx.Request.Context(), "phone_login_sms_delivery_failed",
			"request_id", ctx.GetString("request_id"))
	case err != nil:
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
		return
	}
	replyJSON(ctx, http.StatusAccepted, gin.H{
		"challenge_token": result.ChallengeToken, "expires_in_seconds": 300,
	})
}

// The only endpoint allowed to promote an anonymous phone challenge to the
// same full DB session and JWT used by the email/password login.
func (server *Server) loginPhoneVerify(ctx *gin.Context) {
	var req phoneLoginVerifyRequest
	if !bindJSON(ctx, &req, 8192) {
		return
	}
	if ok, retry := server.authLimiter.allow("phone-login-challenge", fingerprint(req.ChallengeToken),
		ratePolicy{burst: 10, window: 15 * time.Minute}, time.Now()); !ok {
		authRateLimitExceeded(ctx, retry)
		return
	}
	if server.authentication == nil {
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
		return
	}
	result, err := server.authentication.VerifyPhoneLogin(
		ctx.Request.Context(), req.ChallengeToken, req.Code)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAPIError(ctx, http.StatusUnauthorized, "invalid or expired code")
	case errors.Is(err, auth.ErrSMSUnavailable):
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
	case err != nil:
		writeAPIError(ctx, http.StatusServiceUnavailable, "phone login unavailable")
	default:
		server.respondLoginSession(ctx, result)
	}
}
