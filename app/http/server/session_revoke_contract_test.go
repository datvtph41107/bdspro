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

type revokeRequestKey struct{}

func TestRevokeRefreshTokenErrorAndPersistenceContracts(t *testing.T) {
	for _, tc := range []struct {
		name string
		storeErr error
		wantCode int
		wantBody string
	}{
		{"Success", nil, http.StatusNoContent, ""},
		{"MissingOrAlreadyRevoked", pgx.ErrNoRows, http.StatusUnauthorized, `{"error":"session unavailable"}`},
		{"DatabaseUnavailable", errors.New("secret-db-error"), http.StatusInternalServerError, `{"error":"revoke failed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			raw, payload, err := server.tokenMaker.CreateToken(41, time.Hour, token.TokenTypeRefresh)
			require.NoError(t, err)
			ctx := context.WithValue(context.Background(), revokeRequestKey{}, "request-scope")
			store.EXPECT().BlockSession(gomock.Any(), db.BlockSessionParams{
				ID: payload.ID, AccountID: payload.AccountID, RefreshToken: raw,
			}).DoAndReturn(func(actual context.Context, arg db.BlockSessionParams) (db.Session, error) {
				require.Equal(t, "request-scope", actual.Value(revokeRequestKey{}))
				if tc.storeErr != nil {
					return db.Session{}, tc.storeErr
				}
				return db.Session{ID: arg.ID, AccountID: arg.AccountID, RefreshToken: arg.RefreshToken, IsBlocked: true}, nil
			})
			body, err := json.Marshal(refreshTokenRequest{RefreshToken: raw})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/tokens/revoke", bytes.NewReader(body)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
			if tc.wantBody == "" {
				require.Empty(t, rec.Body.String())
			} else {
				require.JSONEq(t, tc.wantBody, rec.Body.String())
			}
			require.NotContains(t, rec.Body.String(), "secret-db-error")
		})
	}
}

func TestRevokeRejectsAccessTokenBeforeStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	raw, _, err := server.tokenMaker.CreateToken(41, time.Hour, token.TokenTypeAccess)
	require.NoError(t, err)
	body, err := json.Marshal(refreshTokenRequest{RefreshToken: raw})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/tokens/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.JSONEq(t, `{"error":"invalid refresh token"}`, rec.Body.String())
	// An unexpected BlockSession call would fail the generated MockStore.
}
