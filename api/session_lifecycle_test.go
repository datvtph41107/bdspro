package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestLogoutAllHTTPContract(t *testing.T) {
	cases := []struct {
		name, cookie, proof             string
		accountErr, logoutAllErr        error
		want, accountCalls, logoutCalls int
	}{
		{"missing_cookie", "", "1", nil, nil, 401, 0, 0},
		{"csrf_denied", "session", "", nil, nil, 403, 0, 0},
		{"invalid_session", "session", "1", auth.ErrUnauthorized, nil, 401, 1, 0},
		{"session_backend_down", "session", "1", errors.New("db down"), nil, 503, 1, 0},
		{"revalidated_session_expired", "session", "1", nil, auth.ErrUnauthorized, 401, 1, 1},
		{"revoke_failure", "session", "1", nil, errors.New("db down"), 503, 1, 1},
		{"successful", "session", "1", nil, nil, 204, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &authStub{accountID: 17, accountErr: tc.accountErr, logoutAllErr: tc.logoutAllErr}
			server := NewServer(nil, fake, nil, nil, true, "")
			req := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: tc.cookie})
			}
			if tc.proof != "" {
				req.Header.Set("X-BDSPro-Request", tc.proof)
			}
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
			require.Equal(t, tc.accountCalls, fake.accountCalls)
			require.Equal(t, tc.logoutCalls, fake.logoutAllCalls)
			if tc.want == 204 {
				require.Empty(t, w.Body.String())
				require.Len(t, w.Result().Cookies(), 1)
				require.Less(t, w.Result().Cookies()[0].MaxAge, 0)
			} else {
				require.Empty(t, w.Result().Cookies(), "do not erase client token if revocation failed")
			}
		})
	}
}
