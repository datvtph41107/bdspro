package user

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/datvtph41107/bdspro/modules/auth"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

const oauth2StateCookie = "bdspro_oidc_state"

// OAuth2Handler handles browser transport; verified identity and account
// binding remain in the auth service and model.
// BrowserLoginCompleter permits the shared, verified OIDC callback to finish
// a browser login without importing a web router into the identity handler.
type BrowserLoginCompleter interface {
	CompleteOIDCLogin(*gin.Context, auth_service.LoginResult, string)
}

type OAuth2Handler struct {
	auth         *auth_service.AuthService
	oauth2       *auth_service.OAuth2Client
	secure       bool
	browserLogin BrowserLoginCompleter
}

func NewOAuth2Handler(auth *auth_service.AuthService, oauth2 *auth_service.OAuth2Client, secure bool, browser ...BrowserLoginCompleter) *OAuth2Handler {
	h := &OAuth2Handler{auth: auth, oauth2: oauth2, secure: secure}
	if len(browser) > 0 {
		h.browserLogin = browser[0]
	}
	return h
}

func (h *OAuth2Handler) stateCookie(c *gin.Context, state string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oauth2StateCookie, Value: state, Path: "/users/oauth2/callback",
		MaxAge: maxAge, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *OAuth2Handler) StartLogin(c *gin.Context) {
	start, err := h.oauth2.Begin(c.Request.Context(), "login", 0, "")
	if err != nil {
		h.unavailable(c, err)
		return
	}
	h.stateCookie(c, start.State, start.MaxAge)
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, start.AuthorizationURL)
}

// StartBrowserLogin uses the same provider callback and PKCE/nonce verification
// as the API login. Only a validated local consent target is kept in Redis.
func (h *OAuth2Handler) StartBrowserLogin(c *gin.Context) {
	if h.browserLogin == nil || h.oauth2 == nil {
		c.Status(http.StatusNotFound)
		return
	}
	next, err := auth.BrowserNextPath(c.Query("next"))
	if err != nil {
		appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_return_path", "invalid consent return path")
		return
	}
	start, err := h.oauth2.BeginBrowserLogin(c.Request.Context(), next)
	if err != nil {
		h.unavailable(c, err)
		return
	}
	h.stateCookie(c, start.State, start.MaxAge)
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, start.AuthorizationURL)
}

func (h *OAuth2Handler) StartLink(c *gin.Context) {
	var req structs.OAuth2LinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	accountID := middleware.AccountID(c)
	if (req.Password == "") == (req.PhoneCode == "") {
		appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_link_proof", "provide exactly one reauthentication method")
		return
	}
	if req.PhoneCode != "" {
		if !h.auth.HasPhone() {
			appctx.Reject(c, http.StatusForbidden, "oauth2.phone_proof_unavailable", "phone reauthentication unavailable")
			return
		}
		if err := h.auth.AuthorizePhoneProof(c.Request.Context(), accountID, req.PhoneCode); err != nil {
			appctx.Error(c, err)
			return
		}
	} else if err := h.auth.CheckPassword(c.Request.Context(), accountID, req.Password); err != nil {
		appctx.Error(c, err)
		return
	}
	start, err := h.oauth2.Begin(c.Request.Context(), "link", accountID, "")
	if err != nil {
		h.unavailable(c, err)
		return
	}
	h.stateCookie(c, start.State, start.MaxAge)
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"authorization_url": start.AuthorizationURL})
}

// StartPhoneBind reauthenticates using an already linked OIDC identity before
// issuing the phone-verification challenge.
func (h *OAuth2Handler) StartPhoneBind(c *gin.Context) {
	if !h.auth.HasPhone() {
		c.Status(http.StatusNotFound)
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Invalid(c, err)
		return
	}
	start, err := h.oauth2.Begin(c.Request.Context(), "phone_bind", middleware.AccountID(c), req.Phone)
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
		appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
		return
	}
	h.stateCookie(c, "", -1)
	pending, err := h.oauth2.Consume(c.Request.Context(), state)
	if err != nil {
		if errors.Is(err, auth_service.ErrOAuth2StateExpired) || errors.Is(err, auth_service.ErrOAuth2StateInvalid) {
			appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "expired or invalid authorization state")
		} else {
			h.unavailable(c, err)
		}
		return
	}
	if c.Query("error") != "" || c.Query("code") == "" {
		appctx.Reject(c, http.StatusUnauthorized, "oauth2.authorization_declined", "authorization declined")
		return
	}
	idToken, err := h.oauth2.Exchange(c.Request.Context(), c.Query("code"), pending.Verifier)
	if errors.Is(err, auth_service.ErrOAuth2AuthorizationFailed) {
		appctx.Reject(c, http.StatusUnauthorized, "oauth2.authorization_failed", "authorization code rejected")
		return
	}
	if err != nil {
		h.unavailable(c, err)
		return
	}
	identity, err := h.oauth2.Verify(c.Request.Context(), idToken, pending.Nonce)
	if err != nil {
		appctx.Reject(c, http.StatusUnauthorized, "oauth2.invalid_identity", "invalid identity assertion")
		return
	}
	switch pending.Purpose {
	case "phone_bind":
		if pending.AccountID <= 0 || !h.auth.HasPhone() {
			appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
			return
		}
		if err := h.auth.StartOAuth2PhoneBind(c.Request.Context(), pending.AccountID, identity, pending.Phone); err != nil {
			appctx.Error(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"phone_verification_started": true})
	case "link":
		if pending.AccountID <= 0 {
			appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
			return
		}
		if err := h.auth.LinkOIDC(c.Request.Context(), pending.AccountID, identity); err != nil {
			appctx.Error(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"linked": true})
	case "login":
		if pending.AccountID != 0 {
			appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "invalid authorization state")
			return
		}
		if pending.ReturnPath != "" && h.browserLogin == nil {
			appctx.Reject(c, http.StatusServiceUnavailable, "oauth2.browser_unavailable", "browser OAuth login unavailable")
			return
		}
		result, err := h.auth.LoginOIDC(c.Request.Context(), identity, c.Request.UserAgent(), c.ClientIP())
		if err != nil {
			appctx.Error(c, err)
			return
		}
		if pending.ReturnPath != "" {
			h.browserLogin.CompleteOIDCLogin(c, result, pending.ReturnPath)
			return
		}
		c.JSON(http.StatusOK, convert.ToLogin(result))
	default:
		appctx.Reject(c, http.StatusBadRequest, "oauth2.invalid_state", "unsupported authorization purpose")
	}
}

func (h *OAuth2Handler) unavailable(c *gin.Context, err error) {
	log.Ctx(c.Request.Context()).Error().
		Str("dependency", "openid_connect").
		Str("error_type", fmt.Sprintf("%T", err)).
		Msg("auth.provider_unavailable")
	appctx.Reject(c, http.StatusServiceUnavailable, "oauth2.unavailable", "identity provider unavailable")
}
