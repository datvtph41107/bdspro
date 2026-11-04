package routers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestOAuth2BrowserLoginAndStateReplay(t *testing.T) {
	_, _, server, _ := authAcceptanceServer(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var nonce string
	var codeChallenge string
	var issuer string
	var clientID = "bdspro-test"
	var email = uuid.NewString() + "@example.test"
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/keys":
			encode := base64.RawURLEncoding.EncodeToString
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{{
					"kty": "RSA", "kid": "key1", "alg": "RS256", "use": "sig",
					"n": encode(key.PublicKey.N.Bytes()),
					"e": encode(big.NewInt(int64(key.PublicKey.E)).Bytes()),
				}},
			})
		case "/token":
			if r.Method != http.MethodPost {
				t.Errorf("unexpected token request method: %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("grant_type") != "authorization_code" ||
				r.Form.Get("client_id") != clientID ||
				r.Form.Get("code") != "one-time-code" ||
				r.Form.Get("code_verifier") == "" {
				t.Error("invalid authorization-code exchange request")
			}
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(digest[:]) != codeChallenge {
				t.Error("PKCE verifier does not match authorization challenge")
			}
			claims := jwt.MapClaims{
				"iss":            issuer,
				"sub":            "external-subject",
				"aud":            clientID,
				"exp":            time.Now().Add(10 * time.Minute).Unix(),
				"iat":            time.Now().Add(-time.Minute).Unix(),
				"nonce":          nonce,
				"email":          email,
				"email_verified": true,
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "key1"
			signed, err := token.SignedString(key)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id_token": signed})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	cfg := server.config
	cfg.OIDC = setting.OAuth2Provider{
		Issuer:           issuer,
		AuthorizationURL: issuer + "/authorize",
		TokenURL:         issuer + "/token",
		JWKSURL:          issuer + "/keys",
		ClientID:         clientID,
		ClientSecret:     "test-secret",
		RedirectURI:      "http://localhost/users/oauth2/callback",
	}
	oauth2, err := auth_service.NewOAuth2Client(cfg.OIDC, server.authRateRedis, provider.Client())
	if err != nil {
		t.Fatal(err)
	}
	server.oauth2Handler = user.NewOAuth2Handler(auth_service.NewOAuth2AuthService(server.store, server.tokenMaker, cfg), oauth2, false)
	server.router = NormalRoutes(server)
	gin.SetMode(gin.TestMode)

	doLogin := func() (int64, string, string) {
		t.Helper()
		start := httptest.NewRecorder()
		server.Handler().ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/users/oauth2/login/start", nil))
		if start.Code != http.StatusFound {
			t.Fatalf("OAuth2 login start: status %d body=%s", start.Code, start.Body.String())
		}
		redirect, err := url.Parse(start.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		nonce = redirect.Query().Get("nonce")
		codeChallenge = redirect.Query().Get("code_challenge")
		state := redirect.Query().Get("state")
		if redirect.Query().Get("code_challenge_method") != "S256" ||
			len(state) != 43 || nonce == "" {
			t.Fatal("missing state/nonce/PKCE")
		}
		cookie := start.Result().Cookies()
		if len(cookie) != 1 || cookie[0].Name != "bdspro_oidc_state" || !cookie[0].HttpOnly {
			t.Fatalf("invalid state cookie: %#v", cookie)
		}
		// The flow callback must consume the browser-bound state once.
		cb := httptest.NewRequest(http.MethodGet, "/users/oauth2/callback?state="+url.QueryEscape(state)+"&code=one-time-code", nil)
		cb.AddCookie(cookie[0])
		result := httptest.NewRecorder()
		server.Handler().ServeHTTP(result, cb)
		if result.Code != http.StatusOK {
			t.Fatalf("OAuth2 callback status %d: %s", result.Code, result.Body.String())
		}
		var body struct {
			User struct {
				AccountID int64 `json:"account_id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(result.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.User.AccountID <= 0 {
			t.Fatalf("OIDC login missing account: %s", result.Body.String())
		}
		replay := httptest.NewRecorder()
		server.Handler().ServeHTTP(replay, cb)
		if replay.Code != http.StatusBadRequest {
			t.Fatalf("replayed state accepted: %d", replay.Code)
		}
		verifier := redirect.Query().Get("code_challenge")
		if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
			t.Fatalf("not a base64url PKCE challenge: %v", err)
		}
		return body.User.AccountID, state, nonce
	}
	firstID, _, _ := doLogin()
	secondID, _, _ := doLogin()
	if firstID != secondID {
		t.Fatalf("the same issuer/subject generated two accounts: %d != %d", firstID, secondID)
	}
}
