package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

type authStub struct {
	registration  auth.Registration
	registerErr   error
	logoutErr     error
	resendErr     error
	registerCalls int
	logoutCalls   int
	resendCalls   int
}

func (s *authStub) Register(context.Context, string, string) (auth.Registration, error) {
	s.registerCalls++
	return s.registration, s.registerErr
}

func (s *authStub) VerifyEmail(context.Context, string) error { return nil }
func (s *authStub) ResendVerification(context.Context, string) error {
	s.resendCalls++
	return s.resendErr
}
func (s *authStub) Login(context.Context, string, string) (auth.LoginResult, error) {
	return auth.LoginResult{}, nil
}
func (s *authStub) AccountFromToken(context.Context, string) (int64, error) {
	return 0, auth.ErrUnauthorized
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
