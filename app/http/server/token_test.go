package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRenewAccessTokenAPI(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "OK", true: "Blocked"}[blocked], func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			raw, payload, err := server.tokenMaker.CreateToken(23, time.Hour, token.TokenTypeRefresh)
			require.NoError(t, err)
			store.EXPECT().GetSession(gomock.Any(), payload.ID).Return(db.Session{
				ID: payload.ID, AccountID: 23, RefreshToken: raw, IsBlocked: blocked, ExpiresAt: time.Now().Add(time.Hour),
			}, nil)
			body, _ := json.Marshal(refreshTokenRequest{RefreshToken: raw})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)
			if blocked {
				require.Equal(t, http.StatusUnauthorized, w.Code)
			} else {
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				var rsp renewAccessTokenResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rsp))
				_, err = server.tokenMaker.VerifyToken(rsp.AccessToken, token.TokenTypeAccess)
				require.NoError(t, err)
			}
		})
	}
}

func TestRenewRejectsAccessToken(t *testing.T) {
	server := newTestServer(t, nil)
	raw, _, err := server.tokenMaker.CreateToken(23, time.Hour, token.TokenTypeAccess)
	require.NoError(t, err)
	body, _ := json.Marshal(refreshTokenRequest{RefreshToken: raw})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRenewSessionFailures(t *testing.T) {
	cases := []struct {
		name   string
		absent bool
		err    error
		change func(*db.Session)
		want   int
	}{
		{"Missing", true, pgx.ErrNoRows, nil, http.StatusUnauthorized},
		{"DatabaseDown", true, errors.New("db down"), nil, http.StatusInternalServerError},
		{"Blocked", false, nil, func(s *db.Session) { s.IsBlocked = true }, http.StatusUnauthorized},
		{"WrongAccount", false, nil, func(s *db.Session) { s.AccountID = 999 }, http.StatusUnauthorized},
		{"WrongToken", false, nil, func(s *db.Session) { s.RefreshToken = "different" }, http.StatusUnauthorized},
		{"Expired", false, nil, func(s *db.Session) { s.ExpiresAt = time.Now().Add(-time.Minute) }, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			raw, payload, err := server.tokenMaker.CreateToken(6, time.Hour, token.TokenTypeRefresh)
			require.NoError(t, err)
			session := db.Session{
				ID: payload.ID, AccountID: 6, RefreshToken: raw, ExpiresAt: time.Now().Add(time.Hour),
			}
			if tc.change != nil {
				tc.change(&session)
			}
			if tc.absent {
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Return(db.Session{}, tc.err)
			} else {
				store.EXPECT().GetSession(gomock.Any(), payload.ID).Return(session, nil)
			}
			body, _ := json.Marshal(refreshTokenRequest{RefreshToken: raw})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
		})
	}
}
