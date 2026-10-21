package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/token"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	server := newTestServer(t, nil)
	access, _, err := server.tokenMaker.CreateToken(19, time.Minute, token.TokenTypeAccess)
	require.NoError(t, err)
	refresh, _, err := server.tokenMaker.CreateToken(19, time.Minute, token.TokenTypeRefresh)
	require.NoError(t, err)
	cases := []struct {
		name, header string
		want         int
	}{
		{"OK", "Bearer " + access, http.StatusOK},
		{"Missing", "", http.StatusUnauthorized},
		{"WrongType", "Bearer " + refresh, http.StatusUnauthorized},
		{"WrongScheme", "Basic " + access, http.StatusUnauthorized},
		{"Malformed", "Bearer", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
		})
	}
}
