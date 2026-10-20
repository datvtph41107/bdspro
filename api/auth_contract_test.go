package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

type authStub struct {
	registration  auth.Registration
	registerErr   error
	verifyErr     error
	loginResult   auth.LoginResult
	loginErr      error
	accountID     int64
	accountErr    error
	accountCalls  int
	accountToken  string
	logoutErr     error
	resendErr     error
	registerCalls int
	verifyCalls   int
	loginCalls    int
	logoutCalls   int
	resendCalls   int
}

func (s *authStub) Register(context.Context, string, string) (auth.Registration, error) {
	s.registerCalls++
	return s.registration, s.registerErr
}

func (s *authStub) VerifyEmail(context.Context, string) error {
	s.verifyCalls++
	return s.verifyErr
}
func (s *authStub) ResendVerification(context.Context, string) error {
	s.resendCalls++
	return s.resendErr
}
func (s *authStub) Login(context.Context, string, string) (auth.LoginResult, error) {
	s.loginCalls++
	return s.loginResult, s.loginErr
}
func (s *authStub) AccountFromToken(_ context.Context, token string) (int64, error) {
	s.accountCalls++
	s.accountToken = token
	return s.accountID, s.accountErr
}
func (s *authStub) Logout(context.Context, string) error {
	s.logoutCalls++
	return s.logoutErr
}

func TestCreateAccountServiceOutcomes(t *testing.T) {
	cases := []struct {
		name   string
		regErr error
		status int
	}{
		{"success", nil, http.StatusCreated},
		{"duplicate_email", auth.ErrEmailUsed, http.StatusConflict},
		{"invalid_registration", auth.ErrInvalid, http.StatusBadRequest},
		{"delivery_failed", auth.ErrMailerUnavailable, http.StatusServiceUnavailable},
		{"unexpected_error", errors.New("database unavailable"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{registration: auth.Registration{AccountID: 17, VerificationToken: "debug-only"}, registerErr: tc.regErr}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"sample@example.test","password":"long-enough-password"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, 1, fake.registerCalls)
			if tc.regErr == nil {
				require.Contains(t, w.Body.String(), `"account_id":17`)
			}
		})
	}
}

func TestLogoutHonorsRevocationFailure(t *testing.T) {
	failure := &authStub{logoutErr: errors.New("database unavailable")}
	server := NewServer(nil, failure, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "session-token"})
	req.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Equal(t, 1, failure.logoutCalls)
	require.Empty(t, w.Result().Cookies(), "do not clear retry token if server-side revoke failed")

	success := &authStub{}
	server = NewServer(nil, success, nil, nil, true, "")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, 1, success.logoutCalls)
	require.NotEmpty(t, w.Result().Cookies())
	require.Less(t, w.Result().Cookies()[0].MaxAge, 0)

	withoutCookie := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	withoutCookie.Header.Set("X-BDSPro-Request", "1")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, withoutCookie)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, 1, success.logoutCalls, "no cookie must not require a backend revoke")
}

func TestResendVerificationDoesNotExposeDeliveryOrAccountStatus(t *testing.T) {
	for _, backendError := range []error{nil, auth.ErrMailerUnavailable, errors.New("database unavailable")} {
		fake := &authStub{resendErr: backendError}
		server := NewServer(nil, fake, nil, nil, true, "")
		req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", strings.NewReader(`{"email":"unknown@example.test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusAccepted, w.Code)
		require.Equal(t, 1, fake.resendCalls)
	}

	fake := &authStub{resendErr: auth.ErrInvalid}
	server := NewServer(nil, fake, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", strings.NewReader(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyEmailServiceOutcomes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"verified", nil, http.StatusNoContent},
		{"invalid_token", auth.ErrInvalidToken, http.StatusBadRequest},
		{"backend_failure", errors.New("database unavailable"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{verifyErr: tc.err}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"verification-token"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)

			require.Equal(t, tc.status, w.Code)
			require.Equal(t, 1, fake.verifyCalls)
			require.Empty(t, w.Result().Cookies())
		})
	}

	t.Run("invalid_request", func(t *testing.T) {
		fake := &authStub{}
		server := NewServer(nil, fake, nil, nil, true, "")
		req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Zero(t, fake.verifyCalls, "invalid input must not reach the service")
	})

	t.Run("service_unavailable", func(t *testing.T) {
		server := NewServer(nil, nil, nil, nil, true, "")
		req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"token":"verification-token"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestLoginServiceOutcomes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"authenticated", nil, http.StatusOK},
		{"invalid_credentials", auth.ErrInvalidCredentials, http.StatusUnauthorized},
		{"unverified_email", auth.ErrUnverified, http.StatusForbidden},
		{"backend_failure", errors.New("database unavailable"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{
				loginResult: auth.LoginResult{
					AccountID: 17,
					Token:     "mock-session",
					ExpiresAt: time.Now().Add(time.Hour),
				},
				loginErr: tc.err,
			}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"sample@example.test","password":"long-enough-password"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-BDSPro-Request", "1")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)

			require.Equal(t, tc.status, w.Code)
			require.Equal(t, 1, fake.loginCalls)
			cookies := w.Result().Cookies()
			if tc.err == nil {
				require.Contains(t, w.Body.String(), `"account_id":17`)
				require.Len(t, cookies, 1)
				require.Equal(t, "bdspro_session", cookies[0].Name)
				require.Equal(t, "mock-session", cookies[0].Value)
				require.True(t, cookies[0].HttpOnly)
				require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
				require.False(t, cookies[0].Secure, "development uses a local HTTP listener")
			} else {
				require.Empty(t, cookies, "login failure must not issue a session cookie")
			}
		})
	}

	t.Run("invalid_request", func(t *testing.T) {
		fake := &authStub{}
		server := NewServer(nil, fake, nil, nil, true, "")
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"sample@example.test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-BDSPro-Request", "1")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Zero(t, fake.loginCalls, "invalid input must not reach the service")
	})

	t.Run("service_unavailable", func(t *testing.T) {
		server := NewServer(nil, nil, nil, nil, true, "")
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"sample@example.test","password":"long-enough-password"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-BDSPro-Request", "1")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestCurrentAccountFailureSemantics(t *testing.T) {
	cases := []struct {
		name       string
		cookie     bool
		accountID  int64
		accountErr error
		status     int
		calls      int
	}{
		{"missing_cookie", false, 0, nil, http.StatusUnauthorized, 0},
		{"expired_session", true, 0, auth.ErrUnauthorized, http.StatusUnauthorized, 1},
		{"database_unavailable", true, 0, errors.New("connection refused"), http.StatusServiceUnavailable, 1},
		{"valid_session", true, 17, nil, http.StatusOK, 1},
		{"zero_account_id", true, 0, nil, http.StatusUnauthorized, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{accountID: tc.accountID, accountErr: tc.accountErr}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			if tc.cookie {
				req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "test-session"})
			}
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.calls, fake.accountCalls)
			if tc.calls > 0 {
				require.Equal(t, "test-session", fake.accountToken)
			}
			if tc.status == http.StatusOK {
				require.Contains(t, w.Body.String(), `"account_id":17`)
				require.Equal(t, 1, fake.accountCalls, "middleware and handler must authenticate only once")
			}
		})
	}
}

func TestLoginProductionCookieContract(t *testing.T) {
	fake := &authStub{loginResult: auth.LoginResult{
		AccountID: 17, Token: "test-session", ExpiresAt: time.Now().Add(time.Hour),
	}}
	server := NewServer(nil, fake, nil, nil, false, "https://app.example.test")
	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"test@example.test","password":"some-password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, w.Result().Cookies(), 1)
	cookie := w.Result().Cookies()[0]
	require.Equal(t, "__Host-bdspro_session", cookie.Name)
	require.Equal(t, "/", cookie.Path)
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}
