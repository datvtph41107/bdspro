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

func TestPhoneOnlyRegistrationStartsWithoutSession(t *testing.T) {
	fake := &authStub{phoneRegistrationStart: auth.PhoneLoginStart{
		ChallengeToken: strings.Repeat("a", 43),
	}}
	server := NewServer(nil, fake, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/phone/register/start",
		strings.NewReader(`{"phone":"+84901234567"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	require.Contains(t, w.Body.String(), "challenge_token")
	require.NotContains(t, w.Body.String(), "account_id")
	require.Empty(t, w.Result().Cookies())
	require.Equal(t, 1, fake.phoneRegistrationStarts)
}

func TestPhoneOnlyRegistrationVerifyControlsSession(t *testing.T) {
	fake := &authStub{phoneRegistrationVerifyErr: auth.ErrInvalidCredentials}
	server := NewServer(nil, fake, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/phone/register/verify",
		strings.NewReader(`{"challenge_token":"test-handle","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, w.Result().Cookies())
	require.Equal(t, 1, fake.phoneRegistrationVerifies)

	fake.phoneRegistrationVerifyErr = nil
	fake.phoneRegistrationVerify = auth.LoginResult{
		AccountID: 77, Token: "opaque-db-session", ExpiresAt: time.Now().Add(time.Hour),
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/phone/register/verify",
		strings.NewReader(`{"challenge_token":"test-handle","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BDSPro-Request", "1")
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"account_id":77`)
	require.Len(t, w.Result().Cookies(), 1)
}

func TestPhoneRegistrationBrowserGate(t *testing.T) {
	fake := &authStub{}
	server := NewServer(nil, fake, nil, nil, true, "")
	for _, target := range []string{"/auth/phone/register/start", "/auth/phone/register/verify"} {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{"phone":"+84901234567"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code, target)
	}
	require.Zero(t, fake.phoneRegistrationStarts)
	require.Zero(t, fake.phoneRegistrationVerifies)
}

func TestPhoneRegistrationExistingNumberDeliveryErrorIndistinguishable(t *testing.T) {
	fake := &authStub{phoneRegistrationStart: auth.PhoneLoginStart{ChallengeToken: strings.Repeat("x", 43)}}
	server := NewServer(nil, fake, nil, nil, true, "")
	makeRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/phone/register/start",
			strings.NewReader(`{"phone":"+84901234567"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-BDSPro-Request", "1")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		return w
	}
	a := makeRequest()
	fake.phoneRegistrationStartErr = auth.ErrPhoneRegistrationDelivery
	b := makeRequest()
	require.Equal(t, http.StatusAccepted, a.Code)
	require.Equal(t, a.Code, b.Code)
	require.Equal(t, a.Body.String(), b.Body.String())
}
