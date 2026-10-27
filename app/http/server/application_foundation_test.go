package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/app/http/binding"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	authrequest "github.com/datvtph41107/bdspro/app/http/requests/auth"
	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthenticatedAccountIDContract(t *testing.T) {
	server := newTestServer(t, nil)

	noToken := httptest.NewRecorder()
	server.Handler().ServeHTTP(noToken, httptest.NewRequest(http.MethodGet, "/users/me", nil))
	require.Equal(t, http.StatusUnauthorized, noToken.Code)

	refresh, _, err := server.tokenMaker.CreateToken(81, time.Hour, token.TokenTypeRefresh)
	require.NoError(t, err)
	wrongTypeReq := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	wrongTypeReq.Header.Set("Authorization", "Bearer "+refresh)
	wrongType := httptest.NewRecorder()
	server.Handler().ServeHTTP(wrongType, wrongTypeReq)
	require.Equal(t, http.StatusUnauthorized, wrongType.Code)

	access, _, err := server.tokenMaker.CreateToken(81, time.Hour, token.TokenTypeAccess)
	require.NoError(t, err)
	validReq := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	validReq.Header.Set("Authorization", "Bearer "+access)
	valid := httptest.NewRecorder()
	server.Handler().ServeHTTP(valid, validReq)
	require.Equal(t, http.StatusOK, valid.Code)
	require.JSONEq(t, `{"account_id":81}`, valid.Body.String())
}

func TestLoginResponseProjectionContract(t *testing.T) {
	id := uuid.New()
	expiredAccess := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)
	expiredRefresh := expiredAccess.Add(24 * time.Hour)
	result := auth.LoginResult{
		Account: db.Account{
			ID: 23, PersonID: 45, Email: "person@example.test",
			PasswordHash: "should-never-be-exposed", IsEmailVerified: true,
		},
		Session:     db.Session{ID: id, AccountID: 23},
		AccessToken: "signed-access", AccessExpiresAt: expiredAccess,
		RefreshToken: "signed-refresh", RefreshExpiresAt: expiredRefresh,
	}
	public := newLoginUserResponse(result)
	require.Equal(t, id.String(), public.SessionID)
	require.Equal(t, result.AccessToken, public.AccessToken)
	require.Equal(t, result.RefreshToken, public.RefreshToken)
	require.Equal(t, expiredAccess, public.AccessTokenExpiresAt)
	require.Equal(t, expiredRefresh, public.RefreshTokenExpiresAt)
	require.Equal(t, int64(23), public.User.AccountID)

	body, err := json.Marshal(public)
	require.NoError(t, err)
	require.NotContains(t, string(body), result.Account.PasswordHash)
	require.NotContains(t, string(body), "password_hash")
	for _, key := range []string{
		"session_id", "access_token", "access_token_expires_at",
		"refresh_token", "refresh_token_expires_at", "user",
	} {
		require.Contains(t, string(body), `"`+key+`"`)
	}
}

type requestContextMarker struct{}

func TestPasswordLoginPropagatesHTTPRequestContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	hash, err := util.HashPassword("strong-password")
	require.NoError(t, err)
	marker := requestContextMarker{}
	value := "request-boundary"

	store.EXPECT().GetAccountByEmail(gomock.Any(), "one@example.test").
		DoAndReturn(func(ctx context.Context, _ string) (db.Account, error) {
			require.Equal(t, value, ctx.Value(marker))
			return db.Account{ID: 51, Email: "one@example.test", PasswordHash: hash}, nil
		})
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
			require.Equal(t, value, ctx.Value(marker))
			return db.Session{ID: arg.ID, AccountID: arg.AccountID, RefreshToken: arg.RefreshToken, ExpiresAt: arg.ExpiresAt}, nil
		})

	server := newTestServer(t, store)
	ctx := context.WithValue(context.Background(), marker, value)
	req := httptest.NewRequest(http.MethodPost, "/users/login",
		strings.NewReader(`{"email":"one@example.test","password":"strong-password"}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestSharedPreJSONBindingKeepsTypedInputAndAbortsBadRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	reached := 0
	router.POST("/test/login", middleware.PreLogin(), func(ctx *gin.Context) {
		reached++
		req := binding.Input[authrequest.LoginBody](ctx, authrequest.LoginKey)
		require.Equal(t, "one@example.test", req.Email)
		require.Equal(t, "strong-password", req.Password)
		ctx.Status(http.StatusNoContent)
	})

	for _, body := range []string{
		"{bad",
		`{"email":"one@example.test"}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.Zero(t, reached, "an invalid body must not reach the next handler")
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test/login",
		strings.NewReader(`{"email":"one@example.test","password":"strong-password"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	require.Equal(t, 1, reached)
}
