package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestAuthTokenBucketRefillsAndBoundsState(t *testing.T) {
	now := time.Unix(10_000, 0)
	l := newAuthLimiter(2)
	policy := ratePolicy{burst: 3, window: 3 * time.Second}
	for i := 0; i < 3; i++ {
		ok, _ := l.allow("login", "one", policy, now)
		require.True(t, ok)
	}
	ok, wait := l.allow("login", "one", policy, now)
	require.False(t, ok)
	require.Equal(t, 1, wait)
	ok, _ = l.allow("login", "one", policy, now.Add(time.Second))
	require.True(t, ok, "one token replenished after one second")
	ok, _ = l.allow("login", "one", policy, now.Add(time.Second))
	require.False(t, ok)

	ok, _ = l.allow("login", "two", policy, now)
	require.True(t, ok)
	ok, _ = l.allow("login", "three", policy, now)
	require.False(t, ok, "never allocate beyond the configured state limit")
	require.Len(t, l.buckets, 2)

	ok, _ = l.allow("login", "three", policy, now.Add(4*time.Second))
	require.True(t, ok, "expired entries are cleared to admit a new peer")
	require.LessOrEqual(t, len(l.buckets), 2)
}

func TestAuthTokenBucketSerializesConcurrentAdmission(t *testing.T) {
	l := newAuthLimiter(128)
	now := time.Unix(20_000, 0)
	policy := ratePolicy{burst: 11, window: time.Hour}
	const attempts = 100
	results := make(chan bool, attempts)
	var workers sync.WaitGroup
	for i := 0; i < attempts; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ok, _ := l.allow("login-account", "same-identity", policy, now)
			results <- ok
		}()
	}
	workers.Wait()
	close(results)
	accepted := 0
	for ok := range results {
		if ok {
			accepted++
		}
	}
	require.Equal(t, 11, accepted)
}

func TestSocketPeerIsCanonicalNotSpoofedProxyHeader(t *testing.T) {
	require.Equal(t, "192.0.2.23", connectionIP("192.0.2.23:4567"))
	require.Equal(t, "192.0.2.23", connectionIP("[::ffff:192.0.2.23]:4567"))
	require.Equal(t, "2001:db8::1", connectionIP("[2001:db8::1]:2345"))
	require.Equal(t, "unknown", connectionIP("unparseable-socket-peer"))
	require.Equal(t, "unknown", connectionIP(""))
}

func loginAttempt(t *testing.T, server *Server, email, remote string, attempt int) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":"any-password"}`, email)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.RemoteAddr = remote
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", (attempt%200)+1))
	req.Header.Set("Forwarded", fmt.Sprintf("for=198.51.100.%d", (attempt%200)+1))
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	return w
}

func assertRateLimitResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	wait, err := strconv.Atoi(w.Header().Get("Retry-After"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, wait, 1)
	require.NotEmpty(t, w.Header().Get("X-Request-ID"))
	var response publicErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "too_many_requests", response.Error.Code)
	require.Equal(t, w.Header().Get("X-Request-ID"), response.Error.RequestID)
	require.NotContains(t, w.Body.String(), "192.0.2.")
	require.NotContains(t, w.Body.String(), "example.test")
}

func TestLoginRateLimitSeparatesSocketIPFromForwardedHeaders(t *testing.T) {
	fake := &authStub{loginErr: auth.ErrInvalidCredentials}
	server := NewServer(nil, fake, nil, nil, true, "")
	for i := 0; i < 30; i++ {
		w := loginAttempt(t, server, fmt.Sprintf("person-%d@example.test", i),
			"192.0.2.8:3434", i)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	}
	w := loginAttempt(t, server, "new@example.test", "192.0.2.8:9999", 555)
	assertRateLimitResponse(t, w)
	require.Equal(t, 30, fake.loginCalls,
		"IP bucket must reject before Argon2id/database Auth Service")
	require.Equal(t, http.StatusUnauthorized,
		loginAttempt(t, server, "different@example.test", "192.0.2.9:3434", 1).Code,
		"another socket IP has its own bucket")
}

func TestLoginRateLimitStopsDistributedGuessingOfSameIdentifier(t *testing.T) {
	fake := &authStub{loginErr: auth.ErrInvalidCredentials}
	server := NewServer(nil, fake, nil, nil, true, "")
	for i := 0; i < 12; i++ {
		w := loginAttempt(t, server, " Victim@Example.Test ",
			fmt.Sprintf("192.0.2.%d:8080", i+20), i)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	}
	w := loginAttempt(t, server, "victim@example.test", "192.0.2.99:9090", 13)
	assertRateLimitResponse(t, w)
	require.Equal(t, 12, fake.loginCalls, "per-identifier bucket rejects even with rotating IPs")
	require.Equal(t, http.StatusUnauthorized,
		loginAttempt(t, server, "other@example.test", "192.0.2.99:9090", 14).Code)
}

func TestAuthRoutesHaveIndependentLimits(t *testing.T) {
	fake := &authStub{loginErr: auth.ErrInvalidCredentials}
	server := NewServer(nil, fake, nil, nil, true, "")
	cases := []struct {
		path    string
		body    string
		limit   int
		success int
	}{
		{"/auth/register", `{"email":"new@example.test","password":"verylongpassword"}`, 12, http.StatusCreated},
		{"/auth/resend-verification", `{"email":"new@example.test"}`, 12, http.StatusAccepted},
		{"/auth/verify-email", `{"token":"random-token"}`, 30, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			for i := 0; i < tc.limit; i++ {
				req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				req.RemoteAddr = "203.0.113.10:4532"
				w := httptest.NewRecorder()
				server.Handler().ServeHTTP(w, req)
				require.Equal(t, tc.success, w.Code)
			}
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.RemoteAddr = "203.0.113.10:7777"
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			assertRateLimitResponse(t, w)
		})
	}
	require.Equal(t, 12, fake.registerCalls)
	require.Equal(t, 12, fake.resendCalls)
	require.Equal(t, 30, fake.verifyCalls)
}

func TestAuthPreflightDoesNotUsePostLoginQuota(t *testing.T) {
	server := NewServer(nil, &authStub{}, nil, nil, true, "http://example.com")
	for i := 0; i < 40; i++ {
		req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type,x-bdspro-request")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	}
	require.Empty(t, server.authLimiter.buckets, "preflight must not consume Auth POST buckets")
}
