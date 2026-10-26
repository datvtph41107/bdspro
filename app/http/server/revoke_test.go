package server

import (
	"bytes"
	"context"
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

func TestRevokeRefreshToken(t *testing.T) {
	tests := []struct {
		name     string
		storeErr error
		want     int
	}{
		{name: "OK", want: http.StatusNoContent},
		{name: "AlreadyBlocked", storeErr: pgx.ErrNoRows, want: http.StatusUnauthorized},
		{name: "DatabaseError", storeErr: errors.New("db unavailable"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			raw, payload, err := server.tokenMaker.CreateToken(42, time.Hour, token.TokenTypeRefresh)
			require.NoError(t, err)
			store.EXPECT().BlockSession(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, arg db.BlockSessionParams) (db.Session, error) {
					require.Equal(t, payload.ID, arg.ID)
					require.Equal(t, payload.AccountID, arg.AccountID)
					require.Equal(t, raw, arg.RefreshToken)
					return db.Session{ID: payload.ID, IsBlocked: true}, tt.storeErr
				},
			)
			body, err := json.Marshal(refreshTokenRequest{RefreshToken: raw})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/tokens/revoke", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			require.Equal(t, tt.want, w.Code, w.Body.String())
		})
	}
}

func TestRevokeRejectsAccessToken(t *testing.T) {
	server := newTestServer(t, nil)
	raw, _, err := server.tokenMaker.CreateToken(42, time.Hour, token.TokenTypeAccess)
	require.NoError(t, err)
	body, err := json.Marshal(refreshTokenRequest{RefreshToken: raw})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/tokens/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRevokeInvalidRequest(t *testing.T) {
	server := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/tokens/revoke", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHealth(t *testing.T) {
	server := newTestServer(t, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
