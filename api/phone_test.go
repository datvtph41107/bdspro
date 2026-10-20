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

func TestPhoneVerificationHTTPFlow(t *testing.T) {
	cases := []struct {
		name, path, body, proof string
		accountErr, serviceErr  error
		status                  int
		phoneCalls, verifyCalls int
	}{
		{"start_success", "/auth/phone/start", `{"phone":"+84901234567"}`, "1", nil, nil, 202, 1, 0},
		{"verify_success", "/auth/phone/verify", `{"phone":"+84901234567","code":"123456"}`, "1", nil, nil, 204, 0, 1},
		{"start_csrf_denied", "/auth/phone/start", `{"phone":"+84901234567"}`, "", nil, nil, 403, 0, 0},
		{"verify_missing_code", "/auth/phone/verify", `{"phone":"+84901234567"}`, "1", nil, nil, 400, 0, 0},
		{"start_throttle", "/auth/phone/start", `{"phone":"+84901234567"}`, "1", nil, auth.ErrPhoneThrottle, 429, 1, 0},
		{"verify_invalid", "/auth/phone/verify", `{"phone":"+84901234567","code":"111111"}`, "1", nil, auth.ErrPhoneCode, 400, 0, 1},
		{"verify_in_use", "/auth/phone/verify", `{"phone":"+84901234567","code":"111111"}`, "1", nil, auth.ErrPhoneInUse, 409, 0, 1},
		{"start_no_sms", "/auth/phone/start", `{"phone":"+84901234567"}`, "1", nil, auth.ErrSMSUnavailable, 503, 1, 0},
		{"session_invalid", "/auth/phone/start", `{"phone":"+84901234567"}`, "1", auth.ErrUnauthorized, nil, 401, 0, 0},
		{"session_db_error", "/auth/phone/start", `{"phone":"+84901234567"}`, "1", errors.New("db"), nil, 503, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &authStub{accountID: 17, accountErr: tc.accountErr,
				phoneStartErr: tc.serviceErr, phoneVerifyErr: tc.serviceErr}
			server := NewServer(nil, stub, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: "sample"})
			if tc.proof != "" {
				req.Header.Set("X-BDSPro-Request", tc.proof)
			}
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.phoneCalls, stub.phoneStartCalls)
			require.Equal(t, tc.verifyCalls, stub.phoneVerifyCalls)
			require.NotContains(t, w.Body.String(), "+84901234567")
		})
	}
}
