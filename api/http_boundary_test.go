package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Prove the actual Server.Handler boundary: CORS wraps Gin, rather than
// treating CORS as another Gin middleware or an authentication mechanism.
func TestHTTPBoundaryCORSWrapsGin(t *testing.T) {
	server := newTestServer(t, nil)
	server.config.AllowedOrigins = []string{"http://localhost:3000"}
	handler := server.Handler()

	t.Run("allowed origin enters Gin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		res := httptest.NewRecorder()

		handler.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		require.Equal(t, "http://localhost:3000", res.Header().Get("Access-Control-Allow-Origin"))
		require.NotEmpty(t, res.Header().Get("X-Request-ID"))
	})

	t.Run("CORS preflight completes before Gin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/users/login", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		res := httptest.NewRecorder()

		handler.ServeHTTP(res, req)

		require.Equal(t, http.StatusNoContent, res.Code)
		require.Equal(t, "http://localhost:3000", res.Header().Get("Access-Control-Allow-Origin"))
		// Gin request ID middleware did not run on the preflight response.
		require.Empty(t, res.Header().Get("X-Request-ID"))
	})

	t.Run("CORS is not backend authorization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "https://unapproved.example")
		res := httptest.NewRecorder()

		handler.ServeHTTP(res, req)

		// A non-browser client may still invoke a public HTTP endpoint.
		require.Equal(t, http.StatusOK, res.Code)
		require.Empty(t, res.Header().Get("Access-Control-Allow-Origin"))
		require.NotEmpty(t, res.Header().Get("X-Request-ID"))
	})
}
