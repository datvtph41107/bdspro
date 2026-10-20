package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestCredentialedCORSPreflightContract(t *testing.T) {
	cases := []struct {
		name, origin, method, headers string
		status                       int
	}{
		{"allowed", "https://app.example.test", "POST", "Content-Type, Idempotency-Key, X-BDSPro-Request", 204},
		{"allowed_case_insensitive_headers", "https://app.example.test", "DELETE", "x-bdspro-request,content-type", 204},
		{"same_origin", "http://example.com", "POST", "content-type", 204},
		{"disallowed_origin", "https://attacker.example.test", "POST", "content-type", 403},
		{"null_origin", "null", "POST", "content-type", 403},
		{"missing_origin", "", "POST", "content-type", 403},
		{"disallowed_method", "https://app.example.test", "PATCH", "content-type", 403},
		{"disallowed_header", "https://app.example.test", "POST", "authorization", 403},
		{"empty_header_token", "https://app.example.test", "POST", "content-type,", 403},
	}
	s := NewServer(nil, nil, nil, nil, true, "https://app.example.test")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/organizations/7/sources", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			req.Header.Set("Access-Control-Request-Method", tc.method)
			req.Header.Set("Access-Control-Request-Headers", tc.headers)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Contains(t, w.Result().Header.Values("Vary"), "Origin")
			require.Contains(t, w.Result().Header.Values("Vary"), "Access-Control-Request-Method")
			require.Contains(t, w.Result().Header.Values("Vary"), "Access-Control-Request-Headers")
			if tc.status == 204 {
				require.Equal(t, tc.origin, w.Header().Get("Access-Control-Allow-Origin"))
				require.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				require.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-BDSPro-Request")
				require.Equal(t, "300", w.Header().Get("Access-Control-Max-Age"))
				require.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"), "allow methods required")
			} else {
				require.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
				if tc.origin == "https://attacker.example.test" || tc.origin == "null" || tc.origin == "" {
					require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
					require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
				}
			}
		})
	}
}

func TestCORSDoesNotReflectUnknownOriginOrUseWildcard(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, true, "https://app.example.test")
	for _, tc := range []struct {
		origin string
		allow  bool
	}{
		{"https://app.example.test", true},
		{"http://example.com", true},
		{"https://unknown.example.test", false},
		{"null", false},
		{"*", false},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health?token=unlogged", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Result().Header.Values("Vary"), "Origin")
			if tc.allow {
				require.Equal(t, tc.origin, w.Header().Get("Access-Control-Allow-Origin"))
				require.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				require.Equal(t, "X-Request-ID", w.Header().Get("Access-Control-Expose-Headers"))
			} else {
				require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
				require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
			}
		})
	}
}

func TestBrowserCSRFGateForSessionCookie(t *testing.T) {
	cases := []struct {
		name, origin, fetchSite, proof string
		status                        int
		calls                         int
	}{
		{"no_browser_proof", "", "", "", 403, 0},
		{"sibling_site_without_origin", "", "same-site", "", 403, 0},
		{"cross_site_even_with_allowed_origin", "https://app.example.test", "cross-site", "1", 403, 0},
		{"attacker_origin", "https://evil.example.test", "", "1", 403, 0},
		{"opaque_origin", "null", "", "1", 403, 0},
		{"configured_origin", "https://app.example.test", "same-site", "", 204, 1},
		{"api_same_origin", "http://example.com", "same-origin", "", 204, 1},
		{"fetch_metadata_same_origin", "", "same-origin", "", 204, 1},
		{"custom_non_simple_header", "", "", "1", 204, 1},
		{"invalid_custom_header", "", "", "other", 403, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &authStub{}
			s := NewServer(nil, a, nil, nil, true, "https://app.example.test")
			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "private-session"})
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}
			if tc.proof != "" {
				req.Header.Set("X-BDSPro-Request", tc.proof)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.calls, a.logoutCalls, "untrusted requests must not invoke auth service")
		})
	}
}

func TestLoginRequiresOriginOrNonSimpleRequestHeader(t *testing.T) {
	cases := []struct {
		name, origin, proof, fetchSite string
		status                        int
		calls                         int
	}{
		{"no_proof", "", "", "", 403, 0},
		{"foreign_origin", "https://evil.example.test", "", "", 403, 0},
		{"allowlisted_origin", "https://app.example.test", "", "same-site", 200, 1},
		{"same_origin", "http://example.com", "", "same-origin", 200, 1},
		{"custom_header", "", "1", "", 200, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &authStub{loginResult: auth.LoginResult{
				AccountID: 17, Token: "sample-token", ExpiresAt: time.Now().Add(time.Hour),
			}}
			s := NewServer(nil, a, nil, nil, true, "https://app.example.test")
			req := httptest.NewRequest(http.MethodPost, "/auth/login",
				strings.NewReader(`{"email":"user@example.test","password":"passphrase"}`))
			req.Header.Set("Content-Type", "application/json")
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.proof != "" {
				req.Header.Set("X-BDSPro-Request", tc.proof)
			}
			if tc.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.calls, a.loginCalls)
		})
	}
}

func TestLegacyTokenFreeAPIRequestsRemainAvailable(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader(`{"email":"example@example.test","password":"long-enough-password"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"no-Cookie API use keeps service-level behavior, not a CSRF 403")
}
