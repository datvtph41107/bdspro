package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Uses only the disposable CI PostgreSQL instance; neither a local dev volume
// nor production can be selected by an unchecked environment URL.
func TestSessionIssueRenewRevokeLifecycleWithPostgres(t *testing.T) {
	dsn := os.Getenv("BUSINESS_TEST_DB_URL")
	if os.Getenv("GITHUB_ACTIONS") != "true" || dsn == "" {
		t.Skip("requires GitHub Actions disposable PostgreSQL")
	}
	target, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", target.Hostname())
	require.Equal(t, "5432", target.Port())
	require.Equal(t, "/bdspro", target.Path)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var dbName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&dbName))
	require.Equal(t, "bdspro", dbName)

	store := db.NewStore(pool)
	account, err := store.RegisterTx(ctx, db.RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-hash",
	})
	require.NoError(t, err)
	server := newTestServer(t, store)
	result, err := auth.IssueSession(ctx, store, server.tokenMaker, server.config, account, "session-client", "198.51.100.40")
	require.NoError(t, err)

	stored, err := store.GetSession(ctx, result.Session.ID)
	require.NoError(t, err)
	require.False(t, stored.IsBlocked)
	require.Equal(t, account.ID, stored.AccountID)
	require.Equal(t, result.RefreshToken, stored.RefreshToken)
	require.WithinDuration(t, result.RefreshExpiresAt, stored.ExpiresAt, time.Microsecond, "PostgreSQL stores timestamps at microsecond precision")
	refreshPayload, err := server.tokenMaker.VerifyToken(result.RefreshToken, token.TokenTypeRefresh)
	require.NoError(t, err)
	require.Equal(t, stored.ID, refreshPayload.ID)

	request := func(path string, raw string) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(refreshTokenRequest{RefreshToken: raw})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		return w
	}
	renewed := request("/tokens/renew_access", result.RefreshToken)
	require.Equal(t, http.StatusOK, renewed.Code, renewed.Body.String())
	var renewResult renewAccessTokenResponse
	require.NoError(t, json.Unmarshal(renewed.Body.Bytes(), &renewResult))
	payload, err := server.tokenMaker.VerifyToken(renewResult.AccessToken, token.TokenTypeAccess)
	require.NoError(t, err)
	require.Equal(t, account.ID, payload.AccountID)
	require.True(t, renewResult.AccessTokenExpiresAt.After(time.Now()))

	revoked := request("/tokens/revoke", result.RefreshToken)
	require.Equal(t, http.StatusNoContent, revoked.Code, revoked.Body.String())
	stored, err = store.GetSession(ctx, result.Session.ID)
	require.NoError(t, err)
	require.True(t, stored.IsBlocked)
	require.Equal(t, http.StatusUnauthorized, request("/tokens/renew_access", result.RefreshToken).Code)
	require.Equal(t, http.StatusUnauthorized, request("/tokens/revoke", result.RefreshToken).Code)

	// Revoke affects the durable refresh session; previously minted access
	// JWTs are intentionally not checked against the session table.
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+result.AccessToken)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var me map[string]int64
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &me))
	require.Equal(t, account.ID, me["account_id"])
}
