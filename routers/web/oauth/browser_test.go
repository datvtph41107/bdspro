package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const browserTestID = "123e4567-e89b-42d3-a456-426614174000"
const browserTestOrigin = "https://auth.example.test"

type browserStore struct {
	db.Store
	session db.Session
	err     error
}

func (s *browserStore) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	if s.err != nil {
		return db.Session{}, s.err
	}
	if s.session.ID != id {
		return db.Session{}, pgx.ErrNoRows
	}
	return s.session, nil
}

type browserOAuthStore struct {
	db.OAuthServerStore
	client      db.OauthClient
	version     int64
	approvals   int
	lastAccount int64
}

func (s *browserOAuthStore) GetOAuthClient(_ context.Context, id uuid.UUID) (db.OauthClient, error) {
	if id != s.client.ID {
		return db.OauthClient{}, db.ErrOAuthClientMissing
	}
	return s.client, nil
}
func (s *browserOAuthStore) GetOAuthConsentVersion(_ context.Context, _ int64, _ uuid.UUID) (int64, error) {
	return s.version, nil
}
func (s *browserOAuthStore) ApproveOAuthAuthCode(_ context.Context, _ uuid.UUID, account int64,
	_, _, _ string, _ []byte, _ time.Time, version int64, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return db.ErrOAuthSessionInactive
	}
	if version != s.version {
		return db.ErrOAuthConsentStale
	}
	s.lastAccount = account
	s.approvals++
	s.version++
	return nil
}

func newBrowserTest(t *testing.T) (*web.Router, *browserStore, *browserOAuthStore, authcrypto.Maker) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	sid := uuid.MustParse(browserTestID)
	store := &browserStore{session: db.Session{
		ID: sid, AccountID: 42, ExpiresAt: time.Now().Add(time.Hour),
	}}
	oauthStore := &browserOAuthStore{client: db.OauthClient{
		ID: uuid.New(), Name: "Partner integration", RedirectUri: "https://client.example.test/callback",
	}}
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := oauth_service.NewConsentProofs(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	codec, err := authcrypto.NewBrowserSessionCodec(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	service := oauth_service.NewService(oauthStore)
	browser, err := NewBrowser(Config{Codec: codec, PublicOrigin: browserTestOrigin})
	if err != nil {
		t.Fatal(err)
	}
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexter())
	r.GinEngine().Use(func(c *gin.Context) {
		ctx := db.WithStore(c.Request.Context(), store)
		ctx = authcrypto.WithMaker(ctx, maker)
		ctx = oauth_service.WithService(ctx, service, proofs)
		ctx = WithBrowser(ctx, browser)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Get("/oauth/login", LoginPage)
	r.Post("/oauth/login", Login)
	r.Get("/oauth/authorize", Authenticate, AuthorizePreview)
	r.Post("/oauth/authorize", Authenticate, Authorize)
	r.Post("/oauth/logout", RequireBrowserSession, Logout)
	return r, store, oauthStore, maker
}

func oauthForm(app *browserOAuthStore) url.Values {
	sum := sha256.Sum256([]byte(strings.Repeat("x", 43)))
	return url.Values{
		"client_id":     {app.client.ID.String()},
		"redirect_uri":  {app.client.RedirectUri},
		"response_type": {"code"}, "scope": {"profile:read"},
		"state":                 {"0123456789abcdef"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
}
func req(t *testing.T, r http.Handler, method, path string, data url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body *strings.Reader
	if data == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(data.Encode())
	}
	request := httptest.NewRequest(method, browserTestOrigin+path, body)
	request.Header.Set("Accept", "text/html")
	if method == http.MethodPost {
		request.Header.Set("Origin", browserTestOrigin)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Sec-Fetch-Site", "same-origin")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, request)
	return rec
}
func hiddenValue(t *testing.T, body, name string) string {
	t.Helper()
	re := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]*)"`)
	match := re.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("missing hidden %s: %.300s", name, body)
	}
	return html.UnescapeString(match[1])
}
func cookieNamed(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("missing cookie %s: %+v", name, rec.Result().Cookies())
	return nil
}

func TestBrowserLoginConsentAndRevokedSession(t *testing.T) {
	r, store, app, maker := newBrowserTest(t)
	form := oauthForm(app)
	next := "/oauth/authorize?" + form.Encode()
	page := req(t, r, http.MethodGet, "/oauth/login?next="+url.QueryEscape(next), nil)
	if page.Code != http.StatusOK {
		t.Fatalf("login page %d: %s", page.Code, page.Body.String())
	}
	loginNonce := cookieNamed(t, page, "__Host-bdspro-login")
	if !loginNonce.Secure || !loginNonce.HttpOnly {
		t.Fatal("unsafe login cookie")
	}
	csrf := hiddenValue(t, page.Body.String(), "csrf")
	badLogin := req(t, r, http.MethodPost, "/oauth/login", url.Values{
		"next": {next}, "email": {"user@example.test"}, "password": {"good-password"},
	}, loginNonce)
	if badLogin.Code != http.StatusForbidden {
		t.Fatalf("login accepted missing CSRF: %d", badLogin.Code)
	}
	openRedirect := req(t, r, http.MethodPost, "/oauth/login", url.Values{
		"next": {"https://evil.example.test/"}, "csrf": {csrf},
	}, loginNonce)
	if openRedirect.Code != http.StatusBadRequest {
		t.Fatalf("browser login accepted untrusted return URL: %d", openRedirect.Code)
	}
	// The pure HTTP/consent fixture starts from a codec-issued browser cookie.
	// Password verification and actual session creation are covered by the
	// disposable-Postgres browser integration test, not a production login mock.
	codec, err := authcrypto.NewBrowserSessionCodec(strings.Repeat("s", 32))
	if err != nil { t.Fatal(err) }
	token, err := codec.IssueCookie(store.session.ID.String(), time.Now(), store.session.ExpiresAt)
	if err != nil { t.Fatal(err) }
	sessionCookie := authcrypto.BrowserSessionCookie(token, true)
	if !sessionCookie.Secure || !sessionCookie.HttpOnly || sessionCookie.Domain != "" {
		t.Fatalf("unsafe browser cookie: %+v", sessionCookie)
	}
	preview := req(t, r, http.MethodGet, next, nil, sessionCookie)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "Partner integration") {
		t.Fatalf("HTML preview=%d %s", preview.Code, preview.Body.String())
	}
	policy := preview.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "form-action 'self' https://client.example.test;") ||
		preview.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid consent CSP or cache headers: %s, %v", policy, preview.Header())
	}
	approve := url.Values{}
	for k, v := range form {
		approve[k] = append([]string(nil), v...)
	}
	approve.Set("approve", "true")
	approve.Set("csrf", hiddenValue(t, preview.Body.String(), "csrf"))
	approve.Set("consent_version", hiddenValue(t, preview.Body.String(), "consent_version"))
	approve.Set("consent_proof", hiddenValue(t, preview.Body.String(), "consent_proof"))
	tampered := url.Values{}
	for k, v := range approve {
		tampered[k] = append([]string(nil), v...)
	}
	tampered.Set("state", "changed_state_0123456789")
	if rec := req(t, r, http.MethodPost, "/oauth/authorize", tampered, sessionCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("tampered OAuth proof accepted: %d", rec.Code)
	}
	noCSRF := url.Values{}
	for k, v := range approve {
		noCSRF[k] = append([]string(nil), v...)
	}
	noCSRF.Del("csrf")
	if rec := req(t, r, http.MethodPost, "/oauth/authorize", noCSRF, sessionCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("missing consent CSRF accepted: %d", rec.Code)
	}
	wrongOriginReq := httptest.NewRequest(http.MethodPost, browserTestOrigin+"/oauth/authorize", strings.NewReader(approve.Encode()))
	wrongOriginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wrongOriginReq.Header.Set("Origin", "https://evil.example.test")
	wrongOriginReq.AddCookie(sessionCookie)
	wrongOrigin := httptest.NewRecorder()
	r.ServeHTTP(wrongOrigin, wrongOriginReq)
	if wrongOrigin.Code != http.StatusForbidden {
		t.Fatalf("cross-origin consent accepted: %d", wrongOrigin.Code)
	}
	if app.approvals != 0 {
		t.Fatalf("unverified consent wrote grant: %d", app.approvals)
	}
	approved := req(t, r, http.MethodPost, "/oauth/authorize", approve, sessionCookie)
	if approved.Code != http.StatusFound || app.approvals != 1 || app.lastAccount != 42 {
		t.Fatalf("approval=%d approvals=%d account=%d body=%s", approved.Code, app.approvals, app.lastAccount, approved.Body.String())
	}
	redirect, err := url.Parse(approved.Header().Get("Location"))
	if err != nil || redirect.Host != "client.example.test" || redirect.Query().Get("state") != form.Get("state") || redirect.Query().Get("code") == "" {
		t.Fatalf("wrong callback: %s %v", approved.Header().Get("Location"), err)
	}
	store.session.IsBlocked = true
	stale := req(t, r, http.MethodGet, next, nil, sessionCookie)
	if stale.Code != http.StatusSeeOther || !strings.HasPrefix(stale.Header().Get("Location"), "/oauth/login?") {
		t.Fatalf("revoked browser cookie was accepted: %d", stale.Code)
	}
	again := req(t, r, http.MethodPost, "/oauth/authorize", approve, sessionCookie)
	if again.Code != http.StatusUnauthorized || app.approvals != 1 {
		t.Fatalf("replayed consent with revoked session: %d", again.Code)
	}
	// No fallback from an invalid explicit Bearer header to a valid browser cookie.
	store.session.IsBlocked = false
	explicit := httptest.NewRequest(http.MethodGet, browserTestOrigin+next, nil)
	explicit.Header.Set("Authorization", "Bearer invalid")
	explicit.Header.Set("Accept", "text/html")
	explicit.AddCookie(sessionCookie)
	expRec := httptest.NewRecorder()
	r.ServeHTTP(expRec, explicit)
	if expRec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid bearer fell back to cookie: %d", expRec.Code)
	}
	bearer, _, err := maker.CreateSessionAccessToken(42, time.Minute, store.session.ID)
	if err != nil {
		t.Fatal(err)
	}
	apiReq := httptest.NewRequest(http.MethodGet, browserTestOrigin+next, nil)
	apiReq.Header.Set("Authorization", "Bearer "+bearer)
	apiReq.Header.Set("Accept", "application/json")
	apiRec := httptest.NewRecorder()
	r.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusOK {
		t.Fatalf("JWT API regression: %d %s", apiRec.Code, apiRec.Body.String())
	}
	var decoded struct {
		Proof string `json:"consent_proof"`
	}
	if err := json.Unmarshal(apiRec.Body.Bytes(), &decoded); err != nil || decoded.Proof == "" {
		t.Fatalf("Bearer API preview lost proof: %s %v", apiRec.Body.String(), err)
	}
	store.err = errors.New("postgres offline")
	offline := req(t, r, http.MethodGet, next, nil, sessionCookie)
	if offline.Code != http.StatusServiceUnavailable {
		t.Fatalf("DB failure disguised as logout: %d", offline.Code)
	}
}

func TestCompleteOIDCLoginIssuesBrowserSessionNotJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &browserStore{session: db.Session{
		ID: uuid.MustParse(browserTestID), AccountID: 55, ExpiresAt: time.Now().Add(time.Hour),
	}}
	codec, err := authcrypto.NewBrowserSessionCodec(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	browser, err := NewBrowser(Config{Codec: codec, PublicOrigin: browserTestOrigin, OIDCEnabled: true})
	if err != nil { t.Fatal(err) }
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexter())
	r.GinEngine().Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(WithBrowser(c.Request.Context(), browser))
		c.Next()
	})
	r.Get("/oidc/done", func(ctx *appctx.APIContext, w http.ResponseWriter) {
		CompleteOIDCLogin(ctx, w, auth_service.LoginResult{Session: store.session,
			AccessToken: "jwt-must-not-appear", RefreshToken: "refresh-must-not-appear"},
			"/oauth/authorize?state=abcdef1234567890")
	})
	rec := req(t, r, http.MethodGet, "/oidc/done", nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/oauth/authorize?state=abcdef1234567890" {
		t.Fatalf("browser completion failed: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if strings.Contains(rec.Body.String(), "jwt-must-not-appear") ||
		strings.Contains(rec.Header().Get("Set-Cookie"), "refresh-must-not-appear") {
		t.Fatal("server leaked an internal JWT to browser")
	}
	cookie := cookieNamed(t, rec, "__Host-bdspro-browser")
	if _, err := codec.ReadCookie(cookie.Value, time.Now()); err != nil {
		t.Fatalf("invalid browser completion cookie: %v", err)
	}
	bad := web.NewRouter()
	bad.AfterRouting(appctx.APIContexter())
	bad.GinEngine().Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(WithBrowser(c.Request.Context(), browser))
		c.Next()
	})
	bad.Get("/oidc/done", func(ctx *appctx.APIContext, w http.ResponseWriter) {
		CompleteOIDCLogin(ctx, w, auth_service.LoginResult{Session: store.session}, "https://evil.example/")
	})
	badRec := req(t, bad, http.MethodGet, "/oidc/done", nil)
	if badRec.Code != http.StatusBadRequest || badRec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("external OIDC return path accepted: %d", badRec.Code)
	}
}
