package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func httpLogEntry(t *testing.T, data []byte) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	require.Len(t, lines, 1, "exactly one request event")
	var entry map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	require.Equal(t, "http_request", entry["msg"])
	require.GreaterOrEqual(t, entry["duration_ms"].(float64), 0.0)
	return entry
}

func loggingTestServer(logs *bytes.Buffer, authentication Authentication) *Server {
	s := NewServer(nil, authentication, nil, nil, true, "")
	s.logger = slog.New(slog.NewJSONHandler(logs, nil))
	return s
}

func TestRequestLoggingSanitizesRouteAndRejectsSpoofedID(t *testing.T) {
	var logs bytes.Buffer
	s := loggingTestServer(&logs, nil)
	req := httptest.NewRequest(http.MethodGet, "/organizations/123/sources/789?token=query-canary", nil)
	req.Header.Set("X-Request-ID", "spoofed-correlation-id")
	req.Header.Set("Authorization", "Bearer authorization-canary")
	req.Header.Set("Referer", "https://example.test/?secret=referer-canary")
	req.Header.Set("X-Forwarded-For", "192.0.2.90")
	req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "cookie-canary"})
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	entry := httpLogEntry(t, logs.Bytes())
	require.Equal(t, "/organizations/:id/sources/:sourceID", entry["route"])
	require.Equal(t, "GET", entry["method"])
	require.Equal(t, float64(http.StatusUnauthorized), entry["status"])
	id := w.Header().Get("X-Request-ID")
	require.Regexp(t, "^[0-9a-f]{32}$", id)
	require.NotEqual(t, "spoofed-correlation-id", id)
	require.Equal(t, id, entry["request_id"])
	for _, secret := range []string{"query-canary", "cookie-canary", "authorization-canary",
		"referer-canary", "spoofed-correlation-id", "192.0.2.90", "/organizations/123/sources/789"} {
		require.NotContains(t, logs.String(), secret)
	}
}

func TestRequestLoggingSuccessForbiddenAndNotFound(t *testing.T) {
	cases := []struct {
		name, method, path, fetchSite, route string
		status                               int
	}{
		{"health", http.MethodGet, "/health", "", "/health", http.StatusOK},
		{"missing_auth", http.MethodGet, "/auth/me?secret=hidden", "", "/auth/me", http.StatusUnauthorized},
		{"cross_site", http.MethodPost, "/auth/logout", "cross-site", "/auth/logout", http.StatusForbidden},
		{"unmatched", http.MethodGet, "/user-controlled-hidden-path?token=hidden", "", "unmatched", http.StatusNotFound},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			s := loggingTestServer(&logs, nil)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			entry := httpLogEntry(t, logs.Bytes())
			require.Equal(t, tc.route, entry["route"])
			require.Equal(t, float64(tc.status), entry["status"])
			require.Equal(t, false, entry["panic_recovered"])
			id := w.Header().Get("X-Request-ID")
			require.Regexp(t, "^[0-9a-f]{32}$", id)
			require.Equal(t, id, entry["request_id"])
			require.False(t, seen[id])
			seen[id] = true
			require.NotContains(t, logs.String(), "hidden")
		})
	}
}

func TestRequestLoggingOmitsBodyAndRecoveredPanicDetails(t *testing.T) {
	var logs bytes.Buffer
	s := loggingTestServer(&logs, &authStub{})
	s.router.GET("/recovery-test", func(_ *gin.Context) { panic("panic-payload-canary") })
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email",
		strings.NewReader(`{"token":"verification-body-canary"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "/auth/verify-email", httpLogEntry(t, logs.Bytes())["route"])
	require.NotContains(t, logs.String(), "verification-body-canary")

	logs.Reset()
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/recovery-test", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	entry := httpLogEntry(t, logs.Bytes())
	require.Equal(t, true, entry["panic_recovered"])
	require.NotContains(t, logs.String(), "panic-payload-canary")
	require.NotContains(t, w.Body.String(), "panic-payload-canary")
}

func TestResendLoggingDoesNotExposeEmailOrBackendError(t *testing.T) {
	var logs bytes.Buffer
	backend := &authStub{resendErr: errors.New("provider failed for private-email-canary@example.test")}
	s := loggingTestServer(&logs, backend)
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification",
		strings.NewReader(`{"email":"private-email-canary@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	require.Equal(t, 1, backend.resendCalls)
	lines := bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n"))
	require.Len(t, lines, 2, "one redacted backend event plus one request event")
	var backendEvent, requestEvent map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &backendEvent))
	require.NoError(t, json.Unmarshal(lines[1], &requestEvent))
	require.Equal(t, "verification_resend_failed", backendEvent["msg"])
	require.Equal(t, "http_request", requestEvent["msg"])
	require.Equal(t, w.Header().Get("X-Request-ID"), backendEvent["request_id"])
	require.Equal(t, backendEvent["request_id"], requestEvent["request_id"])
	require.NotContains(t, logs.String(), "private-email-canary@example.test")
	require.NotContains(t, logs.String(), "provider failed")
}
