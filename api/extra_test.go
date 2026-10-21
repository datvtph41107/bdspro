package api

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
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLoginSessionInsertFailure(t *testing.T) {
	hash, err := util.HashPassword("strong-password")
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAccountByEmail(gomock.Any(), "abc@example.test").Return(
		db.Account{ID: 5, Email: "abc@example.test", PasswordHash: hash}, nil,
	)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(
		db.Session{}, errors.New("postgres-secret-do-not-leak"),
	)
	server := newTestServer(t, store)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewBufferString(
		`{"email":"abc@example.test","password":"strong-password"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "postgres-secret")
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
			body, _ := json.Marshal(renewAccessTokenRequest{RefreshToken: raw})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
		})
	}
}
