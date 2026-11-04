package user

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/routers/api/response"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

func (g *GoogleHandler) Callback(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	state := ctx.Query("state")
	cookie, err := ctx.Request.Cookie(googleOAuthCookie)
	if err != nil || state == "" || len(state) != 43 || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		response.Reject(ctx, http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
		return
	}
	g.stateCookie(ctx, "", -1)
	pending, err := g.flow.Consume(ctx.Request.Context(), state)
	if errors.Is(err, auth_service.ErrGoogleOAuthStateExpired) {
		response.Reject(ctx, http.StatusBadRequest, "google_state_expired", "expired Google authorization state")
		return
	}
	if errors.Is(err, auth_service.ErrGoogleOAuthStateInvalid) {
		response.Reject(ctx, http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
		return
	}
	if err != nil {
		googleUnavailable(ctx, err)
		return
	}
	if ctx.Query("error") != "" || ctx.Query("code") == "" {
		response.Reject(ctx, http.StatusUnauthorized, "google_authorization_declined", "Google authorization declined")
		return
	}
	idToken, err := g.flow.Exchange(ctx.Request.Context(), ctx.Query("code"), pending.Verifier)
	if errors.Is(err, auth_service.ErrGoogleOAuthProviderUnavailable) {
		googleUnavailable(ctx, err)
		return
	}
	if err != nil {
		response.Reject(ctx, http.StatusUnauthorized, "google_authorization_failed", "Google authorization failed")
		return
	}
	identity, err := auth_service.VerifyGoogleIDToken(ctx.Request.Context(), idToken, g.config.GoogleClientID, pending.Nonce)
	if err != nil {
		response.Reject(ctx, http.StatusUnauthorized, "google_invalid_identity", "invalid Google identity")
		return
	}

	if pending.Purpose == "phone_bind" && pending.AccountID > 0 && g.phone != nil {
		if err := g.phone.StartGoogleBind(
			ctx.Request.Context(), pending.AccountID, identity.Subject, pending.Phone,
		); err != nil {
			response.Error(ctx, err)
			return
		}
		ctx.JSON(http.StatusAccepted, gin.H{"phone_verification_started": true})
		return
	}
	if pending.Purpose == "link" && pending.AccountID > 0 {
		if err := g.auth.Link(ctx.Request.Context(), pending.AccountID, identity.Subject); err != nil {
			response.Error(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"linked": true})
		return
	}
	if pending.Purpose != "login" || pending.AccountID != 0 {
		response.Reject(ctx, http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
		return
	}
	result, err := g.auth.Login(ctx.Request.Context(), identity, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		response.Error(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToLogin(result))
}

// googleUnavailable records a bounded diagnostic without logging OAuth codes,
// access tokens, provider response bodies or raw network error messages.
func googleUnavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "google_oauth").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("auth.provider_unavailable")
	response.Reject(c, http.StatusServiceUnavailable, "google_unavailable", "Google sign-in unavailable")
}
