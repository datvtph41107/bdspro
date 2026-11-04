package user

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

const oauth2StateCookie = "bdspro_oidc_state"

// OAuth2Handler handles browser transport; verified identity and account
// binding remain in the auth service and model.
type OAuth2Handler struct {
	auth   *auth_service.OAuth2AuthService
	flow   *auth_service.OAuth2Flow
	secure bool
}

func NewOAuth2Handler(auth *auth_service.OAuth2AuthService, flow *auth_service.OAuth2Flow, secure bool) *OAuth2Handler {
	return &OAuth2Handler{auth: auth, flow: flow, secure: secure}
}

func (h *OAuth2Handler) stateCookie(c *gin.Context, state string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oauth2StateCookie, Value: state, Path: "/users/oauth2/callback",
		MaxAge: maxAge, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *OAuth2Handler) StartLogin(c *gin.Context) {
	start, err := h.flow.Begin(c.Request.Context(), "login", 0, "")
	if err != nil {
		h.unavailable(c, err)
		return
	}
	h.stateCookie(c, start.State, start.MaxAge)
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, start.AuthorizationURL)
}

func (h *OAuth2Handler) StartLink(c *gin.Context) {
	var req structs.OAuth2LinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Invalid(c, err)
		return
	}
	accountID := middleware.AccountID(c)
	if err := h.auth.CheckPassword(c.Request.Context(), accountID, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	start, err := h.flow.Begin(c.Request.Context(), "link", accountID, "")
	if err != nil {
		h.unavailable(c, err)
		return
	}
	h.stateCookie(c, start.State, start.MaxAge)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"authorization_url": start.AuthorizationURL})
}

func (h *OAuth2Handler) Callback(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	state := c.Query("state")
	cookie, err := c.Request.Cookie(oauth2StateCookie)
	if err != nil || len(state) != 43 ||
		subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		response.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
		return
	}
	h.stateCookie(c, "", -1)
	pending, err := h.flow.Consume(c.Request.Context(), state)
	if err != nil {
		if errors.Is(err, auth_service.ErrOAuth2StateExpired) || errors.Is(err, auth_service.ErrOAuth2StateInvalid) {
			response.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "expired or invalid authorization state")
		} else {
			h.unavailable(c, err)
		}
		return
	}
	if c.Query("error") != "" || c.Query("code") == "" {
		response.Reject(c, http.StatusUnauthorized, "oauth2.authorization_declined", "authorization declined")
		return
	}
	idToken, err := h.flow.Exchange(c.Request.Context(), c.Query("code"), pending.Verifier)
	if errors.Is(err, auth_service.ErrOAuth2AuthorizationFailed) {
		response.Reject(c, http.StatusUnauthorized, "oauth2.authorization_failed", "authorization code rejected")
		return
	}
	if err != nil {
		h.unavailable(c, err)
		return
	}
	identity, err := h.flow.Verify(c.Request.Context(), idToken, pending.Nonce)
	if err != nil {
		response.Reject(c, http.StatusUnauthorized, "oauth2.invalid_identity", "invalid identity assertion")
		return
	}
	switch pending.Purpose {
	case "link":
		if pending.AccountID <= 0 {
			response.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
			return
		}
		if err := h.auth.Link(c.Request.Context(), pending.AccountID, identity); err != nil {
			response.Error(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"linked": true})
	case "login":
		if pending.AccountID != 0 {
			response.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
			return
		}
		result, err := h.auth.Login(c.Request.Context(), identity, c.Request.UserAgent(), c.ClientIP())
		if err != nil {
			response.Error(c, err)
			return
		}
		c.JSON(http.StatusOK, convert.ToLogin(result))
	default:
		response.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "unsupported authorization purpose")
	}
}

func (h *OAuth2Handler) unavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "openid_connect").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("auth.provider_unavailable")
	response.Reject(c, http.StatusServiceUnavailable, "oauth2.unavailable", "identity provider unavailable")
}
