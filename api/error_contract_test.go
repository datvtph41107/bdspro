package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type publicErrorResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func checkPublicError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) publicErrorResponse {
	t.Helper()
	require.Equal(t, status, w.Code)
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var payload publicErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, code, payload.Error.Code)
	require.NotEmpty(t, payload.Error.Message)
	require.Regexp(t, "^[0-9a-f]{32}$", payload.Error.RequestID)
	require.Equal(t, w.Header().Get("X-Request-ID"), payload.Error.RequestID)
	return payload
}

func TestPublicErrorContractAcrossHTTPBoundaries(t *testing.T) {
	failure := errors.New("postgres: password=server-secret-canary@example.test")
	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		origin      string
		proof       bool
		cookie      bool
		accountID   int64
		accountErr  error
		registerErr error
		status      int
		code        string
	}{
		{"invalid_json", "POST", "/auth/register", "{", "", false, false, 0, nil, nil, 400, "invalid_request"},
		{"session_missing", "GET", "/auth/me", "", "", false, false, 0, nil, nil, 401, "unauthorized"},
		{"cross_site_login", "POST", "/auth/login", "{}", "https://evil.example.test", true, false, 0, nil, nil, 403, "forbidden"},
		{"unknown_route", "GET", "/private/path?credential=secret-canary", "", "", false, false, 0, nil, nil, 404, "not_found"},
		{"method_not_allowed", "POST", "/health", "", "", false, false, 0, nil, nil, 405, "method_not_allowed"},
		{"email_conflict", "POST", "/auth/register", `{"email":"test@example.test","password":"long-password"}`, "", false, false, 0, nil, auth.ErrEmailUsed, 409, "conflict"},
		{"db_internal_error", "POST", "/auth/register", `{"email":"test@example.test","password":"long-password"}`, "", false, false, 0, nil, failure, 500, "internal_error"},
		{"session_backend_down", "GET", "/auth/me", "", "", false, true, 0, failure, nil, 503, "service_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{accountID: tc.accountID, accountErr: tc.accountErr, registerErr: tc.registerErr}
			server := NewServer(nil, fake, nil, nil, true, "https://app.example.test")
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.proof {
				req.Header.Set("X-BDSPro-Request", "1")
			}
			if tc.cookie {
				req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "private-session"})
			}
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			payload := checkPublicError(t, w, tc.status, tc.code)
			require.NotContains(t, w.Body.String(), "server-secret-canary")
			require.NotContains(t, w.Body.String(), "private-session")
			require.NotContains(t, w.Body.String(), "private/path")
			require.NotContains(t, payload.Error.Message, "postgres")
		})
	}
}

func TestJSONRequestValidationFailsBeforeService(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
		status      int
		code        string
		calls       int
	}{
		{"truncated", "application/json", `{"email":"test"`, 400, "invalid_request", 0},
		{"trailing_document", "application/json", `{"email":"test@example.test","password":"longpass"} {}`, 400, "invalid_request", 0},
		{"json_array", "application/json", `[{"email":"test@example.test"}]`, 400, "invalid_request", 0},
		{"json_null", "application/json", "null", 400, "invalid_request", 0},
		{"missing_required", "application/json", `{"email":"test@example.test"}`, 400, "invalid_request", 0},
		{"unknown_field", "application/json", `{"email":"test@example.test","password":"longpass","password_debug":"secret-canary"}`, 400, "invalid_request", 0},
		{"oversized", "application/json", `{"email":"` + strings.Repeat("a", 8300) + `","password":"longpass"}`, 413, "payload_too_large", 0},
		{"unsupported_media_type", "text/plain", `{"email":"test@example.test","password":"longpass"}`, 415, "unsupported_media_type", 0},
		{"malformed_media_type", "application/json; broken", `{"email":"test@example.test","password":"longpass"}`, 415, "unsupported_media_type", 0},
		{"valid_with_charset", "application/json; charset=utf-8", `{"email":"test@example.test","password":"longpass"}`, 201, "", 1},
		{"legacy_no_content_type", "", `{"email":"test@example.test","password":"longpass"}`, 201, "", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{registration: auth.Registration{AccountID: 17}}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(tc.body))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			if tc.code != "" {
				checkPublicError(t, w, tc.status, tc.code)
				require.NotContains(t, w.Body.String(), "secret-canary")
			} else {
				require.Equal(t, tc.status, w.Code)
				require.Contains(t, w.Body.String(), `"account_id":17`)
			}
			require.Equal(t, tc.calls, fake.registerCalls)
		})
	}
}

func TestRecoveryUsesSanitizedErrorEnvelope(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	server.router.GET("/error-envelope-panic", func(_ *gin.Context) {
		panic("sensitive database DSN and session token")
	})
	req := httptest.NewRequest(http.MethodGet, "/error-envelope-panic", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	checkPublicError(t, w, 500, "internal_error")
	require.NotContains(t, w.Body.String(), "sensitive")
	require.NotContains(t, w.Body.String(), "session token")
}

func TestSuccessfulResponsesKeepExistingPayload(t *testing.T) {
	server := NewServer(nil, nil, nil, nil, true, "")
	health := httptest.NewRecorder()
	server.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, health.Code)
	require.Equal(t, "ok", health.Body.String())

	fake := &authStub{}
	server = NewServer(nil, fake, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("X-BDSPro-Request", "1")
	logout := httptest.NewRecorder()
	server.Handler().ServeHTTP(logout, req)
	require.Equal(t, 204, logout.Code)
	require.Empty(t, logout.Body.String())
}
