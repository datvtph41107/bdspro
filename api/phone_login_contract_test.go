package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestPhoneLoginStartUniformResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"eligible", nil, 202},
		{"provider_sms_failed", auth.ErrPhoneLoginDelivery, 202},
		{"invalid_phone", auth.ErrPhoneInvalid, 400},
		{"sender_not_configured", auth.ErrSMSUnavailable, 503},
		{"database_down", errors.New("db down"), 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{phoneLoginStart: auth.PhoneLoginStart{ChallengeToken: strings.Repeat("x", 43)}, phoneLoginStartErr: tc.err}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/phone/login/start", strings.NewReader(`{"phone":"+84901234567"}`))
			req.Header.Set("X-BDSPro-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
			require.Equal(t, 1, fake.phoneLoginStarts)
			require.Empty(t, w.Result().Cookies(), "start must never issue a session")
			if tc.want == 202 {
				require.Contains(t, w.Body.String(), `"challenge_token":"`)
			}
		})
	}
}

func TestPhoneLoginVerifyCannotCreateSessionFromInvalidCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"bad_or_replayed", auth.ErrInvalidCredentials, 401},
		{"sms_key_missing", auth.ErrSMSUnavailable, 503},
		{"database_down", errors.New("db down"), 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{phoneLoginVerifyErr: tc.err}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/phone/login/verify", strings.NewReader(`{"challenge_token":"challenge","code":"123456"}`))
			req.Header.Set("X-BDSPro-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
			require.Equal(t, 1, fake.phoneLoginVerifies)
			require.Empty(t, w.Result().Cookies())
		})
	}
}

func TestPhoneLoginRequiresBrowserProofAndDoesNotRequireExistingSession(t *testing.T) {
	fake := &authStub{phoneLoginStart: auth.PhoneLoginStart{ChallengeToken: strings.Repeat("x", 43)}}
	server := NewServer(nil, fake, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/phone/login/start", strings.NewReader(`{"phone":"+84901234567"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, 403, w.Code, "anonymous login must use existing browser origin/CSRF rules")
	require.Equal(t, 0, fake.phoneLoginStarts)

	req = httptest.NewRequest(http.MethodPost, "/auth/phone/login/start", strings.NewReader(`{"phone":"+84901234567"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, 202, w.Code, "no authentication should be required to begin phone login")
	require.Equal(t, 1, fake.phoneLoginStarts)
}
