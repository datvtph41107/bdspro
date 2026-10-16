package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestServerHealthAndLegacyRoutes(t *testing.T) {
	legacy := http.NewServeMux()
	legacy.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	})
	server := NewServer(nil, nil, legacy)

	health := httptest.NewRecorder()
	server.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, health.Code)
	require.Equal(t, "ok", health.Body.String())

	ready := httptest.NewRecorder()
	server.Handler().ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	require.Equal(t, http.StatusServiceUnavailable, ready.Code)
	require.Equal(t, "not ready", ready.Body.String())
}

func TestCreateAccountInvalidRequest(t *testing.T) {
	server := NewServer(nil, &auth.Handler{Development: true}, nil)

	for _, body := range []string{
		"{",
		`{"email":"only@example.test"}`,
		`{"email":"bad@example.test","password":"long-enough-password","unexpected":true}`,
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
	server := NewServer(nil, &auth.Handler{Development: true}, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader(`{"email":"an@example.test","password":"long-enough-password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
