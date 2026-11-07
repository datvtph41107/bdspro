package oauth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type userinfoOAuthStore struct {
	db.OAuthServerStore
	err error
}

func (s userinfoOAuthStore) ResolveOAuthAccessToken(context.Context, []byte) (db.OAuthAccess, error) {
	if s.err != nil {
		return db.OAuthAccess{}, s.err
	}
	return db.OAuthAccess{AccountID: 41, Scope: oauth_service.ScopeProfile}, nil
}

type userinfoAccountStore struct {
	db.Store
	err error
}

func (s userinfoAccountStore) GetAccountByID(context.Context, int64) (db.Account, error) {
	if s.err != nil {
		return db.Account{}, s.err
	}
	return db.Account{ID: 41, Email: "verified@example.test", IsEmailVerified: true}, nil
}

func TestUserInfoSeparatesCredentialRejectionFromPostgresFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	unavailable := errors.New("database unavailable")
	cases := []struct {
		name       string
		grantErr   error
		accountErr error
		status     int
		response   string
	}{
		{"expired or revoked token", db.ErrOAuthGrantInvalid, nil, http.StatusUnauthorized, `"error":"invalid_token"`},
		{"token lookup outage", unavailable, nil, http.StatusServiceUnavailable, `"error":"temporarily_unavailable"`},
		{"missing account", nil, pgx.ErrNoRows, http.StatusUnauthorized, `"error":"invalid_token"`},
		{"account lookup outage", nil, unavailable, http.StatusServiceUnavailable, `"error":"temporarily_unavailable"`},
		{"valid token and account", nil, nil, http.StatusOK, `"sub":41`},
	}
	bearer := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(oauth_service.NewService(userinfoOAuthStore{err: tc.grantErr}), userinfoAccountStore{err: tc.accountErr})
			r := gin.New()
			r.GET("/oauth/userinfo", h.UserInfo)
			req := httptest.NewRequest(http.MethodGet, "/oauth/userinfo", nil)
			req.Header.Set("Authorization", "Bearer "+bearer)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			if resp.Code != tc.status || !strings.Contains(resp.Body.String(), tc.response) {
				t.Fatalf("got %d %s; want %d including %q", resp.Code, resp.Body.String(), tc.status, tc.response)
			}
			if resp.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("OAuth userinfo response must not be cached")
			}
			if tc.status == http.StatusOK && (strings.Contains(resp.Body.String(), `"email"`) || strings.Contains(resp.Body.String(), `"email_verified"`) || strings.Contains(resp.Body.String(), "verified@example.test")) {
				t.Fatalf("profile:read leaked contact claims: %s", resp.Body.String())
			}
		})
	}
}
