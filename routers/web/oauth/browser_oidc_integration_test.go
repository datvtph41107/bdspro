package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	api_oauth "github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOIDCBrowserLoginCallbackAndConsentEndToEnd(t *testing.T) {
	raw := os.Getenv("BDS_TEST_DB_URL")
	if raw == "" {
		t.Skip("requires disposable PostgreSQL test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing to write outside disposable database")
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatalf("refusing to write to %s", cfg.ConnConfig.Database)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer redisClient.Close()
	if redisClient.Ping(ctx).Err() != nil {
		t.Skip("Redis required for OIDC state integration")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	email := "oidc-browser-" + uuid.NewString() + "@example.test"
	subject := "subject-" + uuid.NewString()
	var issuer, nonce string
	providerHTTP := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/keys":
			encode := base64.RawURLEncoding.EncodeToString
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{{
					"kty": "RSA", "kid": "browser-test-key", "alg": "RS256", "use": "sig",
					"n": encode(key.PublicKey.N.Bytes()),
					"e": encode(big.NewInt(int64(key.PublicKey.E)).Bytes()),
				}},
			})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("grant_type") != "authorization_code" ||
				r.Form.Get("code") != "test-code" || r.Form.Get("code_verifier") == "" ||
				 r.Form.Get("redirect_uri") != browserTestOrigin+"/users/oauth2/callback" {
				t.Error("provider exchange did not use the configured callback and PKCE")
			}
			tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": issuer, "sub": subject, "aud": "oidc-browser-test",
				"iat": time.Now().Add(-time.Minute).Unix(),
				"exp": time.Now().Add(5*time.Minute).Unix(),
				"nonce": nonce, "email": email, "email_verified": true,
			})
			tok.Header["kid"] = "browser-test-key"
			signed, signErr := tok.SignedString(key)
			if signErr != nil {
				t.Error(signErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id_token": signed})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer providerHTTP.Close()
	issuer = providerHTTP.URL
	provider := setting.OAuth2Provider{
		Issuer: issuer, AuthorizationURL: issuer + "/authorize",
		TokenURL: issuer + "/token", JWKSURL: issuer + "/keys",
		ClientID: "oidc-browser-test", ClientSecret: "test-secret",
		RedirectURI: browserTestOrigin + "/users/oauth2/callback",
	}
	flow, err := auth_service.NewOAuth2Client(provider, redisClient, providerHTTP.Client())
	if err != nil {
		t.Fatal(err)
	}
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
	store := db.NewStore(pool)
	authService := auth_service.NewAuthService(store, maker, setting.Config{
		OIDC: provider, TokenSymmetricKey: secret, AuditKey: strings.Repeat("a", 32),
		AccessTokenDuration: 15*time.Minute, RefreshTokenDuration: 24*time.Hour,
	}, nil)
	clientStore := &browserOAuthStore{client: db.OauthClient{
		ID: uuid.New(), Name: "OIDC browser client", RedirectUri: "https://client.example.test/callback",
	}}
	oauthSvc := oauth_service.NewService(clientStore)
	browser, err := NewHandler(Config{
		Store: store, OAuth: oauthSvc,
		API: api_oauth.NewHandler(oauthSvc, store, proofs), Proofs: proofs,
		Login: authService.Login, Maker: maker, Codec: codec,
		PublicOrigin: browserTestOrigin, OIDCEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	oidc := user.NewOAuth2Handler(authService, flow, true, browser)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/oauth/login/oidc/start", oidc.StartBrowserLogin)
	router.GET("/users/oauth2/callback", oidc.Callback)
	router.GET("/oauth/authorize", browser.Authenticate, browser.AuthorizePreview)

	next := "/oauth/authorize?" + oauthForm(clientStore).Encode()
	start := req(t,router,http.MethodGet,"/oauth/login/oidc/start?next="+url.QueryEscape(next),nil)
	if start.Code != http.StatusFound {
		t.Fatalf("OIDC browser start failed: %d %s",start.Code,start.Body.String())
	}
	stateCookie := cookieNamed(t,start,"bdspro_oidc_state")
	if !stateCookie.HttpOnly || !stateCookie.Secure || stateCookie.Path != "/users/oauth2/callback" {
		t.Fatalf("unsafe OIDC state cookie: %+v", stateCookie)
	}
	location, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Host != strings.TrimPrefix(issuer, "https://") {
		t.Fatalf("OIDC redirect went to wrong host: %s",location)
	}
	state := location.Query().Get("state")
	nonce = location.Query().Get("nonce")
	if state == "" || nonce == "" || state != stateCookie.Value ||
		location.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("incorrect OIDC authorization request: %s",location.RawQuery)
	}

	callbackPath := "/users/oauth2/callback?state="+url.QueryEscape(state)+"&code=test-code"
	callback := req(t,router,http.MethodGet,callbackPath,nil,stateCookie)
	if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != next {
		t.Fatalf("OIDC browser callback failed: %d %s",callback.Code,callback.Body.String())
	}
	if strings.Contains(callback.Body.String(),"access_token") || strings.Contains(callback.Body.String(),"refresh_token") {
		t.Fatal("OIDC callback disclosed internal JWT credentials")
	}
	browserCookie := cookieNamed(t,callback,"__Host-bdspro-browser")
	idText, err := codec.ReadCookie(browserCookie.Value, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.GetSession(ctx,id)
	if err != nil || session.AccountID <= 0 || session.IsBlocked {
		t.Fatalf("OIDC callback did not create live DB session: %+v %v",session,err)
	}
	preview := req(t,router,http.MethodGet,next,nil,browserCookie)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(),"OIDC browser client") {
		t.Fatalf("fresh browser OIDC session cannot review consent: %d %s",preview.Code,preview.Body.String())
	}
	replay := req(t,router,http.MethodGet,callbackPath,nil,stateCookie)
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replayed OIDC state accepted: %d",replay.Code)
	}
}
