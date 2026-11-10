// Package oauth implements browser consent/session transport on top of the
// same context-aware services and OAuth authorization actions used by the API.
package oauth

import (
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
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const browserSessionKey = "oauth_browser_session"
const loginNonceMaxAge = int(authcrypto.BrowserCSRFTokenTTL / time.Second)

// Browser owns only HTTP cookie policy and a validated browser origin.
// SQLC, login, OAuth grants and Redis throttle belong to domain packages.
type Config struct {
	Codec        *authcrypto.BrowserSessionCodec
	PublicOrigin string
	OIDCEnabled  bool
}

type Browser struct {
	Config
	originHost string
	secure     bool
}

func NewBrowser(cfg Config) (*Browser, error) {
	if cfg.Codec == nil {
		return nil, errors.New("browser OAuth requires a cookie codec")
	}
	u, err := url.Parse(cfg.PublicOrigin)
	if err != nil || u == nil || u.Host == "" ||
		(u.Scheme != "https" && u.Scheme != "http") || u.Path != "" ||
		u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errors.New("invalid browser public origin")
	}
	return &Browser{Config: cfg, originHost: u.Host, secure: u.Scheme == "https"}, nil
}

func wantsHTML(ctx *appctx.APIContext) bool {
	return strings.Contains(ctx.CurrentRequest().Header.Get("Accept"), "text/html")
}
func browserHeaders(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Frame-Options", "DENY")
	ctx.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	ctx.Header("Content-Type", "text/html; charset=utf-8")
}

// Chromium checks form-action across redirects: only a registered client
// callback origin may be allowed, never an arbitrary user-supplied host.
func consentHeaders(ctx *appctx.APIContext, redirectURI string) bool {
	u, err := url.Parse(redirectURI)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") ||
		u.Host == "" || u.User != nil {
		ctx.APIError(http.StatusBadRequest, "invalid_request", "invalid request")
		return false
	}
	ctx.Header("Content-Security-Policy",
		"default-src 'none'; form-action 'self' "+u.Scheme+"://"+u.Host+"; base-uri 'none'; frame-ancestors 'none'")
	return true
}

func loginCookieName(secure bool) string {
	if secure {
		return "__Host-bdspro-login"
	}
	return "bdspro_login_dev"
}
func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}
func sameHost(ctx *appctx.APIContext, b *Browser) bool {
	return strings.EqualFold(ctx.CurrentRequest().Host, b.originHost)
}
func validPOST(ctx *appctx.APIContext, b *Browser) bool {
	return sameHost(ctx, b) &&
		authcrypto.VerifyBrowserPostOrigin(ctx.CurrentRequest(), b.PublicOrigin) == nil
}
func browser(ctx *appctx.APIContext) (*Browser, bool) {
	b := FromContext(ctx)
	if b == nil {
		ctx.APIErrorNotFound()
		return nil, false
	}
	return b, true
}

// loadSession never trusts a cookie as account/session authority without a
// PostgreSQL read. Stale cookies are cleared; DB outages fail closed.
func loadSession(ctx *appctx.APIContext, w http.ResponseWriter, b *Browser) (db.Session, bool) {
	cookie, err := ctx.CurrentRequest().Cookie(authcrypto.BrowserCookieName(b.secure))
	if err != nil {
		return db.Session{}, false
	}
	rawID, err := b.Codec.ReadCookie(cookie.Value, time.Now())
	if err != nil {
		clearCookie(w, authcrypto.BrowserCookieName(b.secure), b.secure)
		return db.Session{}, false
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return db.Session{}, false
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
		return db.Session{}, false
	}
	session, err := store.GetSession(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		clearCookie(w, authcrypto.BrowserCookieName(b.secure), b.secure)
		return db.Session{}, false
	}
	if err != nil {
		ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
		return db.Session{}, false
	}
	if session.ID != id || session.AccountID <= 0 || session.IsBlocked || !session.ExpiresAt.After(time.Now()) {
		clearCookie(w, authcrypto.BrowserCookieName(b.secure), b.secure)
		return db.Session{}, false
	}
	return session, true
}

func notAuthenticated(ctx *appctx.APIContext, w http.ResponseWriter) {
	if ctx.IsAborted() {
		return
	}
	if ctx.CurrentRequest().Method == http.MethodGet && wantsHTML(ctx) {
		next, err := authcrypto.BrowserNextPath(ctx.CurrentRequest().URL.RequestURI())
		if err == nil {
			http.Redirect(w, ctx.CurrentRequest(), "/oauth/login?next="+url.QueryEscape(next), http.StatusSeeOther)
			return
		}
	}
	ctx.Header("WWW-Authenticate", "Bearer")
	ctx.APIError(http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
}

func setVerifiedSession(ctx *appctx.APIContext, session db.Session) {
	payload := &authcrypto.Payload{AccountID: session.AccountID,
		TokenType: authcrypto.TokenTypeAccess, SessionID: session.ID}
	ctx.Doer = payload
	ctx.PublishVerifiedActor(auth_model.Actor{AccountID: session.AccountID, SessionID: session.ID})
	ctx.Data[browserSessionKey] = session.ID.String()
}

// Authenticate accepts either a verified bearer JWT or a verified browser
// cookie. An explicitly bad Authorization header cannot fall back to cookie.
func Authenticate(ctx *appctx.APIContext, w http.ResponseWriter) {
	b := FromContext(ctx)
	if b == nil {
		appctx.Authenticate()(ctx)
		if !ctx.IsAborted() {
			appctx.RequireSession()(ctx)
		}
		return
	}
	if ctx.CurrentRequest().Header.Get("Authorization") != "" {
		appctx.Authenticate()(ctx)
		if ctx.IsAborted() {
			return
		}
		appctx.RequireSession()(ctx)
		return
	}
	session, ok := loadSession(ctx, w, b)
	if !ok {
		notAuthenticated(ctx, w)
		return
	}
	setVerifiedSession(ctx, session)
}

// Logout must use browser cookie authority; a bearer JWT alone is insufficient.
func RequireBrowserSession(ctx *appctx.APIContext, w http.ResponseWriter) {
	b, ok := browser(ctx)
	if !ok {
		return
	}
	session, ok := loadSession(ctx, w, b)
	if !ok {
		notAuthenticated(ctx, w)
		return
	}
	setVerifiedSession(ctx, session)
}

// oauthFailure emits an OAuth2 protocol response, not the first-party API
// error envelope. Its body never contains credential or database details.
func oauthFailure(ctx *appctx.APIContext, err error) {
	status, code := http.StatusServiceUnavailable, "temporarily_unavailable"
	switch {
	case errors.Is(err, oauth_service.ErrInvalidRequest),
		errors.Is(err, oauth_service.ErrConsentProofInvalid):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, oauth_service.ErrInvalidClient):
		status, code = http.StatusUnauthorized, "invalid_client"
	case errors.Is(err, oauth_service.ErrAccessDenied):
		status, code = http.StatusForbidden, "access_denied"
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(status, map[string]any{"error": code})
}

// AuthorizePreview renders one service-owned consent decision as HTML or JSON.
// No router package imports another router's OAuth HTTP handlers.
func AuthorizePreview(ctx *appctx.APIContext, w http.ResponseWriter) {
	actor, ok := ctx.VerifiedActor()
	if !ok { ctx.AuthenticationRequired(); return }
	s := oauth_service.ServiceFromContext(ctx)
	if s == nil { ctx.APIErrorNotFound(); return }
	request, err := oauth_service.ParseAuthorizationRequest(ctx.CurrentRequest().URL.Query())
	if err != nil { oauthFailure(ctx, err); return }
	now := time.Now()
	preview, err := s.PreviewConsent(ctx, actor.AccountID, request,
		oauth_service.ProofsFromContext(ctx), now)
	if err != nil { oauthFailure(ctx, err); return }
	b := FromContext(ctx)
	sessionID, _ := ctx.Data[browserSessionKey].(string)
	if b == nil || sessionID == "" || !wantsHTML(ctx) {
		ctx.Header("Cache-Control", "no-store")
		ctx.Header("Referrer-Policy", "no-referrer")
		ctx.Header("X-Frame-Options", "DENY")
		ctx.JSON(http.StatusOK, map[string]any{
			"client_id": preview.Client.ID.String(), "name": preview.Client.Name,
			"redirect_uri": preview.Client.RedirectURI, "scope": preview.Request.Scope,
			"consent_required": true, "consent_version": preview.Request.ConsentVersion,
			"consent_proof": preview.Proof,
		})
		return
	}
	csrf, err := b.Codec.IssueCSRF(sessionID, "oauth-consent", now)
	if err != nil { ctx.Status(http.StatusServiceUnavailable); return }
	logoutCSRF, err := b.Codec.IssueCSRF(sessionID, "oauth-logout", now)
	if err != nil { ctx.Status(http.StatusServiceUnavailable); return }
	browserHeaders(ctx)
	if !consentHeaders(ctx, preview.Client.RedirectURI) { return }
	_ = templates.OAuthConsentPage.Execute(w, struct {
		ClientName, ClientID, RedirectURI, ResponseType, Scope, State string
		CodeChallenge, CodeChallengeMethod, ConsentProof, CSRF, LogoutCSRF string
		ConsentVersion int64
	}{
		ClientName: preview.Client.Name, ClientID: preview.Client.ID.String(),
		RedirectURI: preview.Request.RedirectURI,
		ResponseType: preview.Request.ResponseType, Scope: preview.Request.Scope,
		State: preview.Request.State,
		CodeChallenge: preview.Request.CodeChallenge,
		CodeChallengeMethod: preview.Request.CodeChallengeMethod,
		ConsentProof: preview.Proof, CSRF: csrf, LogoutCSRF: logoutCSRF,
		ConsentVersion: preview.Request.ConsentVersion,
	})
}

// Authorize enforces browser origin/CSRF only for cookie-based approval.
// Both browser and bearer flows share services/oauth's proof and grant checks.
func Authorize(ctx *appctx.APIContext, w http.ResponseWriter) {
	b := FromContext(ctx)
	sessionID, _ := ctx.Data[browserSessionKey].(string)
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if b != nil && sessionID != "" {
		if !validPOST(ctx, b) ||
			!strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			ctx.Status(http.StatusForbidden)
			return
		}
		if err := req.ParseForm(); err != nil ||
			b.Codec.VerifyCSRF(req.PostForm.Get("csrf"), sessionID, "oauth-consent", time.Now()) != nil {
			ctx.Status(http.StatusForbidden)
			return
		}
	}
	actor, ok := ctx.VerifiedActor()
	if !ok { ctx.AuthenticationRequired(); return }
	s := oauth_service.ServiceFromContext(ctx)
	if s == nil { ctx.APIErrorNotFound(); return }
	if err := req.ParseForm(); err != nil {
		oauthFailure(ctx, oauth_service.ErrInvalidRequest)
		return
	}
	decision, approved, proof, err := oauth_service.ParseConsentDecision(req.PostForm)
	if err != nil { oauthFailure(ctx, err); return }
	payload, ok := ctx.SignedPrincipal()
	if !ok { ctx.AuthenticationRequired(); return }
	redirect, err := s.DecideConsent(ctx, actor.AccountID, payload.SessionID,
		decision, approved, proof, oauth_service.ProofsFromContext(ctx), time.Now())
	if err != nil { oauthFailure(ctx, err); return }
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	http.Redirect(w, req, redirect, http.StatusFound)
}

func LoginPage(ctx *appctx.APIContext, w http.ResponseWriter) {
	b, ok := browser(ctx)
	if !ok {
		return
	}
	if !sameHost(ctx, b) {
		ctx.Status(http.StatusBadRequest)
		return
	}
	next, err := authcrypto.BrowserNextPath(ctx.CurrentRequest().URL.Query().Get("next"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	encoded := base64.RawURLEncoding.EncodeToString(nonce[:])
	csrf, err := b.Codec.IssueCSRF(encoded, "oauth-login", time.Now())
	if err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: loginCookieName(b.secure), Value: encoded, Path: "/",
		HttpOnly: true, Secure: b.secure, SameSite: http.SameSiteLaxMode, MaxAge: loginNonceMaxAge})
	link := ""
	if b.OIDCEnabled {
		link = "/oauth/login/oidc/start?next=" + url.QueryEscape(next)
	}
	browserHeaders(ctx)
	_ = templates.OAuthLoginPage.Execute(w, struct{ Next, CSRF, OIDCStartURL string }{next, csrf, link})
}

func Login(ctx *appctx.APIContext, w http.ResponseWriter) {
	b, ok := browser(ctx)
	if !ok {
		return
	}
	if !validPOST(ctx, b) ||
		!strings.HasPrefix(ctx.CurrentRequest().Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		ctx.Status(http.StatusForbidden)
		return
	}
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if err := req.ParseForm(); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	form := req.PostForm
	next, err := authcrypto.BrowserNextPath(form.Get("next"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	nonce, err := req.Cookie(loginCookieName(b.secure))
	if err != nil || b.Codec.VerifyCSRF(form.Get("csrf"), nonce.Value, "oauth-login", time.Now()) != nil {
		ctx.Status(http.StatusForbidden)
		return
	}
	email, password := form.Get("email"), form.Get("password")
	if len(email) > 254 || len(password) > 1024 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if !security.ThrottleFromContext(ctx).AllowAPI(ctx, "login", email, 30, 10, time.Minute) {
		return
	}
	result, err := auth_service.FromContext(ctx).Login(ctx, email, password, req.UserAgent(), ctx.ClientIP())
	if err != nil {
		if errors.Is(err, auth_model.ErrInvalidCredentials) {
			ctx.Status(http.StatusUnauthorized)
		} else {
			ctx.Status(http.StatusServiceUnavailable)
		}
		return
	}
	completeSessionLogin(ctx, w, b, result, next)
}

func CompleteOIDCLogin(ctx *appctx.APIContext, w http.ResponseWriter, result auth_service.LoginResult, next string) {
	b, ok := browser(ctx)
	if !ok {
		return
	}
	if !b.OIDCEnabled || !sameHost(ctx, b) {
		ctx.Status(http.StatusForbidden)
		return
	}
	if _, err := authcrypto.BrowserNextPath(next); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	completeSessionLogin(ctx, w, b, result, next)
}

func completeSessionLogin(ctx *appctx.APIContext, w http.ResponseWriter, b *Browser, result auth_service.LoginResult, next string) {
	if result.Session.ID == uuid.Nil || result.Session.AccountID <= 0 {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	token, err := b.Codec.IssueCookie(result.Session.ID.String(), time.Now(), result.Session.ExpiresAt)
	if err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(w, authcrypto.BrowserSessionCookie(token, b.secure))
	clearCookie(w, loginCookieName(b.secure), b.secure)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Referrer-Policy", "no-referrer")
	http.Redirect(w, ctx.CurrentRequest(), next, http.StatusSeeOther)
}

// RequireEnabled denies browser-only routes when this server has no browser
// session configuration; generic OAuth authorization still accepts JWTs.
func RequireEnabled(ctx *appctx.APIContext) {
	if FromContext(ctx) == nil {
		ctx.APIErrorNotFound()
	}
}

func Logout(ctx *appctx.APIContext, w http.ResponseWriter) {
	b, ok := browser(ctx)
	if !ok {
		return
	}
	if !validPOST(ctx, b) ||
		!strings.HasPrefix(ctx.CurrentRequest().Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		ctx.Status(http.StatusForbidden)
		return
	}
	req := ctx.CurrentRequest()
	req.Body = http.MaxBytesReader(w, req.Body, 8<<10)
	if err := req.ParseForm(); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	sessionID, _ := ctx.Data[browserSessionKey].(string)
	if b.Codec.VerifyCSRF(req.PostForm.Get("csrf"), sessionID, "oauth-logout", time.Now()) != nil {
		ctx.Status(http.StatusForbidden)
		return
	}
	id, err := uuid.Parse(sessionID)
	if err != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	if err := auth_model.RevokeOwnedSession(ctx, actor.AccountID, id, id); err != nil {
		if errors.Is(err, auth_model.ErrSessionUnavailable) || errors.Is(err, auth_model.ErrSessionNotFound) {
			ctx.Status(http.StatusUnauthorized)
		} else {
			ctx.Status(http.StatusServiceUnavailable)
		}
		return
	}
	clearCookie(w, authcrypto.BrowserCookieName(b.secure), b.secure)
	ctx.Header("Cache-Control", "no-store")
	ctx.Status(http.StatusNoContent)
}
