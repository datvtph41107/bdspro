package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/datvtph41107/bdspro/modules/web"
)

type noGrantStore struct {
	db.OAuthServerStore
}

func oauthHTTPTestRouter(t *testing.T) *web.Router {
	t.Helper()
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	r := oauthTestRouter(oauth_service.NewService(&noGrantStore{}), nil, nil, maker)
	r.Post("/oauth/token", Token)
	r.Post("/oauth/revoke", Revoke)
	r.Get("/oauth/userinfo", UserInfo)
	r.Get("/oauth/authorize", appctx.Authenticate(), appctx.RequireSession(), AuthorizePreview)
	r.Post("/oauth/authorize", appctx.Authenticate(), appctx.RequireSession(), Authorize)
	return r
}

func TestOAuth2OnlyCodeAndRefreshGrant(t *testing.T) {
	router := oauthHTTPTestRouter(t)
	for _, tc := range []struct {
		name, contentType, body string
		want                    int
	}{
		{"password grant rejected", "application/x-www-form-urlencoded", "grant_type=password&client_id=abc", http.StatusUnauthorized},
		{"unsupported grant for UUID client", "application/x-www-form-urlencoded", "grant_type=password&client_id=4df5bcd8-2cd2-496d-9ec1-cd4f99d979ce", http.StatusBadRequest},
		{"JSON rejected", "application/json", `{"grant_type":"password"}`, http.StatusBadRequest},
		{"malformed form rejected", "application/x-www-form-urlencoded", "%", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("got status %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("token error leaked cache header: %v", rec.Header())
			}
		})
	}
}

func TestOAuth2AuthorizeIsAuthenticatedAndUserInfoRejectsInternalJWT(t *testing.T) {
	router := oauthHTTPTestRouter(t)
	for _, path := range []string{"/oauth/authorize", "/oauth/authorize?" + url.Values{"response_type": {"code"}}.Encode()} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthed authorization view returned %d, want 401", rec.Code)
		}
	}
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	internal, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+internal)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("internal JWT was accepted as OAuth access token: %d", rec.Code)
	}
}
