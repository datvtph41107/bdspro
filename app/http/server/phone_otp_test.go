package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type fakeOTPRepo struct{ accountID int64 }

func (r *fakeOTPRepo) AccountIDForPhone(_ context.Context, _ string) (int64, error) {
	return r.accountID, nil
}
func (r *fakeOTPRepo) StartChallenge(_ context.Context, _, _ string, _ int64, _ []byte, _ string) error {
	return nil
}
func (r *fakeOTPRepo) ConsumeChallenge(_ context.Context, _, _ string, _ int64, _ []byte) error {
	return nil
}

func otpTestServer(t *testing.T, store db.Store, environment string) *Server {
	t.Helper()
	cfg := util.Config{
		Environment: environment, TokenSymmetricKey: strings.Repeat("s", 32),
		AccessTokenDuration: 15 * time.Minute, RefreshTokenDuration: 24 * time.Hour,
	}
	server, err := NewServer(cfg, store, testTaskDistributor{})
	require.NoError(t, err)
	return server
}
func TestPhoneOTPRoutesProtectedAndDisabledInProduction(t *testing.T) {
	p := otpTestServer(t, nil, "production")
	require.NoError(t, p.EnablePhoneOTP(&fakeOTPRepo{accountID: 4}))
	req := httptest.NewRequest(http.MethodPost, "/users/phone/login/start",
		bytes.NewBufferString(`{"phone":"+84901234567"}`))
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)

	dev := otpTestServer(t, nil, "development")
	require.Error(t, dev.EnablePhoneOTP(nil))
	require.NoError(t, dev.EnablePhoneOTP(&fakeOTPRepo{accountID: 4}))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/users/me/phone/start",
		bytes.NewBufferString(`{"phone":"+84901234567","password":"secret"}`))
	dev.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
func TestPhoneOTPLoginCreatesBDSProSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	account := db.Account{ID: 17, PersonID: 19, Email: "person@example.test"}
	store.EXPECT().GetAccountByID(gomock.Any(), account.ID).Return(account, nil)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
			require.Equal(t, account.ID, arg.AccountID)
			require.NotEmpty(t, arg.RefreshToken)
			return db.Session{ID: arg.ID, AccountID: arg.AccountID,
				RefreshToken: arg.RefreshToken, ExpiresAt: arg.ExpiresAt}, nil
		},
	)
	server := otpTestServer(t, store, "development")
	require.NoError(t, server.EnablePhoneOTP(&fakeOTPRepo{accountID: 17}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/phone/login/verify",
		bytes.NewBufferString(`{"phone":"+84901234567","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "access_token")
	require.Contains(t, rec.Body.String(), "refresh_token")
	require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
	access, _, err := server.tokenMaker.CreateToken(account.ID, time.Minute, token.TokenTypeAccess)
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	server.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"account_id":17`)
}
