package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerHealthAndReadiness(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	health := httptest.NewRecorder()
	server.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, health.Code)
	require.Equal(t, "ok", health.Body.String())

	ready := httptest.NewRecorder()
	server.Handler().ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	require.Equal(t, http.StatusServiceUnavailable, ready.Code)
}

func TestAllLegacyRoutesNowUseGin(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	want := map[string]bool{
		"POST /auth/register": false,
		"POST /auth/verify-email": false,
		"POST /auth/login": false,
		"GET /auth/me": false,
		"POST /auth/logout": false,
		"POST /organizations/registrations": false,
		"POST /organizations/:id/claims": false,
		"POST /platform/organizations/registrations/:id/approve": false,
		"POST /platform/organizations/claims/:id/approve": false,
		"POST /organizations/:id/invitations": false,
		"POST /organizations/invitations/:id/accept": false,
		"GET /organizations/:id/members": false,
		"POST /organizations/:id/sources": false,
		"GET /organizations/:id/sources/:sourceID": false,
		"POST /organizations/:id/sources/:sourceID/price-reports": false,
		"GET /organizations/:id/sources/:sourceID/price-reports": false,
		"GET /health": false,
		"GET /ready": false,
	}
	for _, route := range server.router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok { want[key] = true }
	}
	for name, found := range want { require.True(t, found, "missing Gin route: %s", name) }
}

func TestCreateAccountInvalidRequest(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	for _, body := range []string{
		"{",
		`{"email":"only@example.test"}`,
		`{"email":"bad@example.test","password":"long-enough-password","unexpected":true}`,
		`{"email":"bad@example.test","password":"long-enough-password"}{"extra":true}`,
	} {
		t.Run(body, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestCreateAccountRejectsCrossSiteRequest(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "https://app.example.test")
	for _, header := range []struct{ name, value string }{
		{"Sec-Fetch-Site", "cross-site"},
		{"Origin", "https://attacker.example.test"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/auth/register",
			strings.NewReader(`{"email":"an@example.test","password":"long-enough-password"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(header.name, header.value)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	}
}

func TestProtectedRoutesRejectMissingSession(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	for _, target := range []string{
		"/auth/me", "/organizations/1/members",
		"/organizations/1/sources/1", "/organizations/1/sources/1/price-reports",
	} {
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		require.Equal(t, http.StatusUnauthorized, w.Code, target)
	}
}
