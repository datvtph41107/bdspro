package auth

import (
	"context"
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

func TestIssueSessionAfterGoogleIdentityVerification(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	maker, err := token.NewJWTMaker(strings.Repeat("s", 32))
	require.NoError(t, err)
	config := util.Config{AccessTokenDuration: 15 * time.Minute, RefreshTokenDuration: 24 * time.Hour}
	account := db.Account{ID: 41, Email: "linked@example.test"}
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
		require.Equal(t, account.ID, arg.AccountID)
		require.NotEmpty(t, arg.RefreshToken)
		require.Equal(t, "test-agent", arg.UserAgent)
		require.True(t, arg.ExpiresAt.After(time.Now()))
		return db.Session{ID: arg.ID, AccountID: arg.AccountID, RefreshToken: arg.RefreshToken, ExpiresAt: arg.ExpiresAt}, nil
	})
	result, err := IssueSession(context.Background(), store, maker, config, account, "test-agent", "127.0.0.1")
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, account.ID, result.Session.AccountID)
	_, err = maker.VerifyToken(result.AccessToken, token.TokenTypeAccess)
	require.NoError(t, err)
	_, err = maker.VerifyToken(result.RefreshToken, token.TokenTypeRefresh)
	require.NoError(t, err)
}
