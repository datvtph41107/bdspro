package oauth

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	api_oauth "github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This test exercises the actual SQLC-backed session and audit revocation,
// not the HTTP mock. It only writes into the explicitly disposable test DB.
func TestBrowserPostgresLoginAndLogoutRevokeSession(t *testing.T) {
	raw := os.Getenv("BDS_TEST_DB_URL")
	if raw == "" {
		t.Skip("requires disposable PostgreSQL 18 test DB")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refuse PostgreSQL writes outside disposable test mode")
	}
	poolCfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if poolCfg.ConnConfig.Database != "bdspro_test" {
		t.Fatalf("refusing browser test writes outside bdspro_test: %s", poolCfg.ConnConfig.Database)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	var personID, accountID int64
	if err := pool.QueryRow(ctx, "INSERT INTO parties DEFAULT VALUES RETURNING id").Scan(&personID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO persons(party_id) VALUES($1)", personID); err != nil {
		t.Fatal(err)
	}
	email := "oauth-browser-"+uuid.NewString()+"@example.test"
	if err := pool.QueryRow(ctx, "INSERT INTO accounts(person_id,email) VALUES($1,$2) RETURNING id", personID, email).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	password := "browser-valid-password"
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO password_credentials(account_id,password_hash) VALUES($1,$2)", accountID, hash); err != nil {
		t.Fatal(err)
	}

	store := db.NewStore(pool)
	secret := strings.Repeat("s", 32)
	maker, err := authcrypto.NewJWTMaker(secret)
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := oauth_service.NewConsentProofs(secret)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := authcrypto.NewBrowserSessionCodec(secret)
	if err != nil {
		t.Fatal(err)
	}
	auth := auth_service.NewAuthService(store, maker, setting.Config{
		TokenSymmetricKey: secret,
		AccessTokenDuration: 15*time.Minute,
		RefreshTokenDuration: 24*time.Hour,
	}, nil)
	oauthStore := &browserOAuthStore{client: db.OauthClient{
		ID: uuid.New(), Name: "Database session client", RedirectUri: "https://client.example.test/callback",
	}}
	oauthSvc := oauth_service.NewService(oauthStore)
	browser, err := NewHandler(Config{
		Store: store, OAuth: oauthSvc,
		API: api_oauth.NewHandler(oauthSvc, store, proofs), Proofs: proofs,
		Maker: maker, Codec: codec, Login: auth.Login,
		PublicOrigin: browserTestOrigin,
	})
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/oauth/login", browser.LoginPage)
	router.POST("/oauth/login", browser.Login)
	router.GET("/oauth/authorize", browser.Authenticate, browser.AuthorizePreview)
	router.POST("/oauth/logout", browser.RequireBrowserSession, browser.Logout)

	next := "/oauth/authorize?"+oauthForm(oauthStore).Encode()
	loginPage := req(t, router, http.MethodGet, "/oauth/login?next="+url.QueryEscape(next), nil)
	if loginPage.Code != http.StatusOK {
		t.Fatalf("login page %d: %s", loginPage.Code, loginPage.Body.String())
	}
	nonce := cookieNamed(t, loginPage, "__Host-bdspro-login")
	login := req(t, router, http.MethodPost, "/oauth/login", url.Values{
		"next": {next}, "csrf": {hiddenValue(t, loginPage.Body.String(), "csrf")},
		"email": {email}, "password": {password},
	}, nonce)
	if login.Code != http.StatusSeeOther {
		t.Fatalf("SQL-backed browser login rejected: %d %s", login.Code, login.Body.String())
	}
	sessionCookie := cookieNamed(t, login, "__Host-bdspro-browser")
	sessionText, err := codec.ReadCookie(sessionCookie.Value, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	sessionID, err := uuid.Parse(sessionText)
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.GetSession(ctx, sessionID)
	if err != nil || session.AccountID != accountID || session.IsBlocked {
		t.Fatalf("wrong login session: %+v, %v", session, err)
	}

	consent := req(t, router, http.MethodGet, next, nil, sessionCookie)
	if consent.Code != http.StatusOK {
		t.Fatalf("consent page rejected valid DB session: %d %s", consent.Code, consent.Body.String())
	}
	logout := req(t, router, http.MethodPost, "/oauth/logout", url.Values{
		"csrf": {hiddenValue(t, consent.Body.String(), "csrf")},
	}, sessionCookie)
	if logout.Code != http.StatusForbidden {
		t.Fatalf("consent CSRF incorrectly authorized logout: %d", logout.Code)
	}
	pageHTML := consent.Body.String()
	logoutStart := strings.Index(pageHTML, `action="/oauth/logout"`)
	if logoutStart == -1 {
		t.Fatal("logout form was not rendered")
	}
	logoutCSRF := hiddenValue(t, pageHTML[logoutStart:], "csrf")
	logout = req(t, router, http.MethodPost, "/oauth/logout", url.Values{
		"csrf": {logoutCSRF},
	}, sessionCookie)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout transaction failed: %d %s", logout.Code, logout.Body.String())
	}
	session, err = store.GetSession(ctx, sessionID)
	if err != nil || !session.IsBlocked {
		t.Fatalf("logout failed to revoke DB session: %+v %v", session, err)
	}
	after := req(t, router, http.MethodGet, next, nil, sessionCookie)
	if after.Code != http.StatusSeeOther {
		t.Fatalf("revoked DB session accepted after logout: %d", after.Code)
	}
}
