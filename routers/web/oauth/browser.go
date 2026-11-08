package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	api_oauth "github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const browserSessionKey = "oauth_browser_session"
const loginNonceMaxAge = int(authcrypto.BrowserCSRFTokenTTL / time.Second)

// Login uses the existing credential service: it must create a normal DB session.
// The result's JWTs must never be exposed to a browser cookie or HTML appctx.
type LoginFunc func(context.Context, string, string, string, string) (auth_service.LoginResult, error)

type Config struct {
	Store        db.Store
	OAuth        *oauth_service.Service
	API          *api_oauth.Handler
	Proofs       *oauth_service.ConsentProofs
	Login        LoginFunc
	Maker        authcrypto.Maker
	Codec        *authcrypto.BrowserSessionCodec
	PublicOrigin string
	Throttle     *security.AuthThrottle
	OIDCEnabled  bool
}

type Handler struct {
	Config
	originHost string
	secure     bool
}

func NewHandler(cfg Config) (*Handler, error) {
	if cfg.Store == nil || cfg.OAuth == nil || cfg.API == nil ||
		cfg.Proofs == nil || cfg.Login == nil || cfg.Maker == nil || cfg.Codec == nil {
		return nil, errors.New("browser OAuth requires store, services and signing keys")
	}
	u, err := url.Parse(cfg.PublicOrigin)
	if err != nil || u == nil || u.Host == "" ||
		(u.Scheme != "https" && u.Scheme != "http") || u.Path != "" ||
		u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errors.New("invalid browser public origin")
	}
	return &Handler{Config: cfg, originHost: u.Host, secure: u.Scheme == "https"}, nil
}

func wantsHTML(c *gin.Context) bool {
	return strings.Contains(c.GetHeader("Accept"), "text/html")
}

func browserHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	c.Header("Content-Type", "text/html; charset=utf-8")
}

// consentHeaders permits only the registered callback origin as a possible
// redirect after the same-origin consent POST. Chromium checks form-action
// against redirects, not just the HTML form action. Never allow arbitrary hosts.
func consentHeaders(c *gin.Context, registeredRedirect string) {
	u, err := url.Parse(registeredRedirect)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self' "+u.Scheme+"://"+u.Host+"; base-uri 'none'; frame-ancestors 'none'")
}

func loginCookieName(secure bool) string {
	if secure {
		return "__Host-bdspro-login"
	}
	return "bdspro_login_dev"
}

func (h *Handler) clearCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: name, Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) sameHost(c *gin.Context) bool {
	// Host is never the authority for deciding an origin. Only the configured
	// value is; Host is checked merely to reject mismatched ingress routing.
	return strings.EqualFold(c.Request.Host, h.originHost)
}

func (h *Handler) validPOST(c *gin.Context) bool {
	return h.sameHost(c) &&
		authcrypto.VerifyBrowserPostOrigin(c.Request, h.PublicOrigin) == nil
}

func (h *Handler) loadSession(c *gin.Context) (db.Session, bool) {
	cookie, err := c.Request.Cookie(authcrypto.BrowserCookieName(h.secure))
	if err != nil {
		return db.Session{}, false
	}
	rawID, err := h.Codec.ReadCookie(cookie.Value, time.Now())
	if err != nil {
		h.clearCookie(c, authcrypto.BrowserCookieName(h.secure))
		return db.Session{}, false
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return db.Session{}, false
	}
	session, err := h.Store.GetSession(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		h.clearCookie(c, authcrypto.BrowserCookieName(h.secure))
		return db.Session{}, false
	}
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return db.Session{}, false
	}
	if session.ID != id || session.AccountID <= 0 || session.IsBlocked || !session.ExpiresAt.After(time.Now()) {
		h.clearCookie(c, authcrypto.BrowserCookieName(h.secure))
		return db.Session{}, false
	}
	return session, true
}

func (h *Handler) notAuthenticated(c *gin.Context) {
	if c.IsAborted() {
		return
	}
	if c.Request.Method == http.MethodGet && wantsHTML(c) {
		next, err := authcrypto.BrowserNextPath(c.Request.URL.RequestURI())
		if err == nil {
			c.Redirect(http.StatusSeeOther, "/oauth/login?next="+url.QueryEscape(next))
			c.Abort()
			return
		}
	}
	appctx.BearerRequired(c)
	c.Abort()
}

// Authenticate preserves the existing Bearer-JWT API contract. An explicitly
// supplied bad Authorization header is NEVER replaced by a browser cookie.
// Browser requests must resolve their DB session on EVERY authorization call.
func (h *Handler) Authenticate(c *gin.Context) {
	if value := c.GetHeader("Authorization"); value != "" {
		h.API.Authenticate(h.Maker)(c)
		return
	}
	session, ok := h.loadSession(c)
	if !ok {
		h.notAuthenticated(c)
		return
	}
	c.Set(appctx.AuthorizationPayloadKey, &authcrypto.Payload{
		AccountID: session.AccountID, TokenType: authcrypto.TokenTypeAccess, SessionID: session.ID,
	})
	c.Set(browserSessionKey, session.ID.String())
	c.Next()
}

func (h *Handler) RequireBrowserSession(c *gin.Context) {
	session, ok := h.loadSession(c)
	if !ok {
		h.notAuthenticated(c)
		return
	}
	c.Set(appctx.AuthorizationPayloadKey, &authcrypto.Payload{
		AccountID: session.AccountID, TokenType: authcrypto.TokenTypeAccess, SessionID: session.ID,
	})
	c.Set(browserSessionKey, session.ID.String())
	c.Next()
}

func (h *Handler) AuthorizePreview(c *gin.Context) {
	sessionID := c.GetString(browserSessionKey)
	if sessionID == "" || !wantsHTML(c) {
		h.API.AuthorizePreview(c)
		return
	}
	request, err := oauth_service.ParseAuthorizationRequest(c.Request.URL.Query())
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	client, version, err := h.OAuth.ReviewConsent(c.Request.Context(), appctx.AccountID(c), request)
	if err != nil {
		if errors.Is(err, oauth_service.ErrInvalidRequest) || errors.Is(err, oauth_service.ErrInvalidClient) {
			c.AbortWithStatus(http.StatusBadRequest)
		} else {
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
		return
	}
	request.ConsentVersion = version
	now := time.Now()
	proof, err := h.Proofs.Issue(oauth_service.ConsentProofFields{
		AccountID:           appctx.AccountID(c),
		ClientID:            request.ClientID.String(),
		RedirectURI:         request.RedirectURI,
		Scope:               request.Scope,
		ResponseType:        request.ResponseType,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
		State:               request.State,
		ConsentVersion:      version,
	}, now)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	csrf, err := h.Codec.IssueCSRF(sessionID, "oauth-consent", now)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	logoutCSRF, err := h.Codec.IssueCSRF(sessionID, "oauth-logout", now)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	browserHeaders(c)
	consentHeaders(c, client.RedirectUri)
	if c.IsAborted() {
		return
	}
	c.Status(http.StatusOK)
	if err := templates.OAuthConsentPage.Execute(c.Writer, struct {
		ClientName, ClientID, RedirectURI, ResponseType, Scope, State      string
		CodeChallenge, CodeChallengeMethod, ConsentProof, CSRF, LogoutCSRF string
		ConsentVersion                                                     int64
	}{
		ClientName: client.Name, ClientID: client.ID.String(),
		RedirectURI: request.RedirectURI, ResponseType: request.ResponseType,
		Scope: request.Scope, State: request.State,
		CodeChallenge:       request.CodeChallenge,
		CodeChallengeMethod: request.CodeChallengeMethod,
		ConsentProof:        proof, CSRF: csrf, LogoutCSRF: logoutCSRF,
		ConsentVersion: version,
	}); err != nil {
		// Never log the sensitive authorization query string.
		_ = err
	}
}

func (h *Handler) Authorize(c *gin.Context) {
	sessionID := c.GetString(browserSessionKey)
	if sessionID != "" {
		if !h.validPOST(c) ||
			!strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
		if err := c.Request.ParseForm(); err != nil ||
			h.Codec.VerifyCSRF(c.Request.PostForm.Get("csrf"), sessionID, "oauth-consent", time.Now()) != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
	h.API.Authorize(c) // Same consent version and signed request proof as the API.
}

func (h *Handler) LoginPage(c *gin.Context) {
	if !h.sameHost(c) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	next, err := authcrypto.BrowserNextPath(c.Query("next"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	encoded := base64.RawURLEncoding.EncodeToString(nonce[:])
	csrf, err := h.Codec.IssueCSRF(encoded, "oauth-login", time.Now())
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: loginCookieName(h.secure), Value: encoded, Path: "/",
		HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
		MaxAge: loginNonceMaxAge,
	})
	link := ""
	if h.OIDCEnabled {
		link = "/oauth/login/oidc/start?next=" + url.QueryEscape(next)
	}
	browserHeaders(c)
	c.Status(http.StatusOK)
	_ = templates.OAuthLoginPage.Execute(c.Writer, struct{ Next, CSRF, OIDCStartURL string }{next, csrf, link})
}

func (h *Handler) Login(c *gin.Context) {
	if !h.validPOST(c) ||
		!strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if err := c.Request.ParseForm(); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	form := c.Request.PostForm
	next, err := authcrypto.BrowserNextPath(form.Get("next"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	loginNonce, err := c.Request.Cookie(loginCookieName(h.secure))
	if err != nil || h.Codec.VerifyCSRF(form.Get("csrf"), loginNonce.Value, "oauth-login", time.Now()) != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	email, password := form.Get("email"), form.Get("password")
	if len(email) > 254 || len(password) > 1024 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if h.Throttle != nil && !h.Throttle.Allow(c, "login", email, 30, 10, time.Minute) {
		c.Abort()
		return
	}
	result, err := h.Config.Login(c.Request.Context(), email, password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, auth_model.ErrInvalidCredentials) {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
		return
	}
	h.completeSessionLogin(c, result, next)
}

// CompleteOIDCLogin finishes an already verified OIDC identity with the
// canonical browser session cookie, without returning any JWT in the appctx.
func (h *Handler) CompleteOIDCLogin(c *gin.Context, result auth_service.LoginResult, next string) {
	if !h.OIDCEnabled || !h.sameHost(c) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if _, err := authcrypto.BrowserNextPath(next); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	h.completeSessionLogin(c, result, next)
}

func (h *Handler) completeSessionLogin(c *gin.Context, result auth_service.LoginResult, next string) {
	if result.Session.ID == uuid.Nil || result.Session.AccountID <= 0 {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	token, err := h.Codec.IssueCookie(result.Session.ID.String(), now, result.Session.ExpiresAt)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(c.Writer, authcrypto.BrowserSessionCookie(token, h.secure))
	h.clearCookie(c, loginCookieName(h.secure))
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusSeeOther, next)
}

func (h *Handler) Logout(c *gin.Context) {
	if !h.validPOST(c) || !strings.HasPrefix(c.ContentType(), "application/x-www-form-urlencoded") {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if err := c.Request.ParseForm(); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	sessionID := c.GetString(browserSessionKey)
	if h.Codec.VerifyCSRF(c.Request.PostForm.Get("csrf"), sessionID, "oauth-logout", time.Now()) != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id, err := uuid.Parse(sessionID)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if err := auth_model.RevokeOwnedSession(c.Request.Context(), h.Store, appctx.AccountID(c), id, id); err != nil {
		if errors.Is(err, auth_model.ErrSessionUnavailable) || errors.Is(err, auth_model.ErrSessionNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
		return
	}
	h.clearCookie(c, authcrypto.BrowserCookieName(h.secure))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
