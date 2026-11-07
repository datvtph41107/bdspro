package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type authorizationHTTPStore struct {
	db.OAuthServerStore
	listedAccount  int64
	revokedAccount int64
	revokedClient  uuid.UUID
}

func (s *authorizationHTTPStore) ListOAuthAuthorizations(_ context.Context, id int64) ([]db.OAuthAuthorizedClient, error) {
	s.listedAccount = id
	return []db.OAuthAuthorizedClient{{ClientID: uuid.MustParse("4df5bcd8-2cd2-496d-9ec1-cd4f99d979ce"), Name: "client app"}}, nil
}
func (s *authorizationHTTPStore) RevokeOAuthAuthorization(_ context.Context, id int64, client uuid.UUID) error {
	s.revokedAccount, s.revokedClient = id, client
	return nil
}

func TestOAuthAuthorizationsHTTPBoundToJWTAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &authorizationHTTPStore{}
	h := NewHandler(oauth_service.NewService(store), nil)
	r := gin.New()
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	group := r.Group("", middleware.Authenticate(maker))
	group.GET("/users/me/oauth/authorizations", h.ListAuthorizations)
	group.DELETE("/users/me/oauth/authorizations/:client_id", h.RevokeAuthorization)
	call := func(method, path, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	client := "4df5bcd8-2cd2-496d-9ec1-cd4f99d979ce"
	if w := call(http.MethodGet, "/users/me/oauth/authorizations", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("public list returned %d", w.Code)
	}
	if w := call(http.MethodDelete, "/users/me/oauth/authorizations/"+client, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("public revoke returned %d", w.Code)
	}
	if store.listedAccount != 0 || store.revokedAccount != 0 {
		t.Fatal("unauthenticated request reached persistence")
	}
	access, _, err := maker.CreateToken(41, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	w := call(http.MethodGet, "/users/me/oauth/authorizations?account_id=999", access)
	if w.Code != http.StatusOK || store.listedAccount != 41 {
		t.Fatalf("list not scoped to JWT account: %d, account=%d, body=%s", w.Code, store.listedAccount, w.Body.String())
	}
	var response struct {
		Authorizations []struct {
			ClientID string `json:"client_id"`
			Name     string `json:"name"`
			Scope    string `json:"scope"`
		} `json:"authorizations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || len(response.Authorizations) != 1 ||
		response.Authorizations[0].ClientID != client || response.Authorizations[0].Scope != oauth_service.ScopeProfile {
		t.Fatalf("unexpected list response %+v; error=%v", response, err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("authorization list is cacheable")
	}
	w = call(http.MethodDelete, "/users/me/oauth/authorizations/not-a-uuid", access)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed client ID returned %d", w.Code)
	}
	w = call(http.MethodDelete, "/users/me/oauth/authorizations/"+client+"?account_id=999", access)
	if w.Code != http.StatusNoContent || store.revokedAccount != 41 || store.revokedClient.String() != client {
		t.Fatalf("revoke not scoped to JWT account: %d account=%d client=%s", w.Code, store.revokedAccount, store.revokedClient)
	}
}
