package auth

import (
	"context"
	"errors"
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

type issueSessionTestMaker struct {
	token.Maker
	failOn  token.TokenType
	failure error
}

func (m issueSessionTestMaker) CreateToken(accountID int64, duration time.Duration, kind token.TokenType) (string, *token.Payload, error) {
	if kind == m.failOn {
		return "", nil, m.failure
	}
	return m.Maker.CreateToken(accountID, duration, kind)
}

func TestIssueSessionPersistsRefreshIdentityAndReturnsCorrectTokenTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := util.Config{
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		TokenSymmetricKey:    strings.Repeat("s", 32),
	}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	require.NoError(t, err)
	account := db.Account{ID: 81, PersonID: 49, Email: "test@example.test", PasswordHash: "not-public"}
	userAgent, clientIP := "integration-client", "198.51.100.21"
	store.EXPECT().CreateSession(ctx, gomock.Any()).
		DoAndReturn(func(received context.Context, params db.CreateSessionParams) (db.Session, error) {
			require.Same(t, ctx, received)
			require.Equal(t, account.ID, params.AccountID)
			require.Equal(t, userAgent, params.UserAgent)
			require.Equal(t, clientIP, params.ClientIp)
			refreshPayload, err := maker.VerifyToken(params.RefreshToken, token.TokenTypeRefresh)
			require.NoError(t, err)
			require.Equal(t, account.ID, refreshPayload.AccountID)
			require.Equal(t, refreshPayload.ID, params.ID)
			require.Equal(t, refreshPayload.ExpiredAt, params.ExpiresAt)
			require.False(t, params.IsBlocked)
			return db.Session{ID: params.ID, AccountID: params.AccountID, RefreshToken: params.RefreshToken, UserAgent: params.UserAgent, ClientIp: params.ClientIp, ExpiresAt: params.ExpiresAt}, nil
		})
	result, err := IssueSession(ctx, store, maker, cfg, account, userAgent, clientIP)
	require.NoError(t, err)
	require.Equal(t, account, result.Account)
	require.Equal(t, result.Session.RefreshToken, result.RefreshToken)
	accessPayload, err := maker.VerifyToken(result.AccessToken, token.TokenTypeAccess)
	require.NoError(t, err)
	refreshPayload, err := maker.VerifyToken(result.RefreshToken, token.TokenTypeRefresh)
	require.NoError(t, err)
	require.Equal(t, account.ID, accessPayload.AccountID)
	require.Equal(t, account.ID, refreshPayload.AccountID)
	require.Equal(t, refreshPayload.ID, result.Session.ID)
	require.Equal(t, accessPayload.ExpiredAt, result.AccessExpiresAt)
	require.Equal(t, refreshPayload.ExpiredAt, result.RefreshExpiresAt)
	require.Greater(t, result.RefreshExpiresAt, result.AccessExpiresAt)
	_, err = maker.VerifyToken(result.AccessToken, token.TokenTypeRefresh)
	require.Error(t, err)
	_, err = maker.VerifyToken(result.RefreshToken, token.TokenTypeAccess)
	require.Error(t, err)
}

func TestIssueSessionStopsBeforePersistenceOnTokenError(t *testing.T) {
	for _, kind := range []token.TokenType{token.TokenTypeAccess, token.TokenTypeRefresh} {
		t.Run(string(kind), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			maker, err := token.NewJWTMaker(strings.Repeat("s", 32))
			require.NoError(t, err)
			want := errors.New("signer unavailable")
			cfg := util.Config{AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
			result, err := IssueSession(context.Background(), store, issueSessionTestMaker{
				Maker: maker, failOn: kind, failure: want,
			}, cfg, db.Account{ID: 81}, "agent", "ip")
			require.ErrorIs(t, err, want)
			require.Empty(t, result.AccessToken)
			require.Empty(t, result.RefreshToken)
			require.Equal(t, LoginResult{}, result)
			// An unexpected CreateSession on the mock fails the test.
		})
	}
}

func TestIssueSessionDoesNotReturnIssuedTokensOnPersistenceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	maker, err := token.NewJWTMaker(strings.Repeat("s", 32))
	require.NoError(t, err)
	want := errors.New("session insert unavailable")
	ctx := context.Background()
	store.EXPECT().CreateSession(ctx, gomock.Any()).Return(db.Session{}, want)
	cfg := util.Config{AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
	result, err := IssueSession(ctx, store, maker, cfg, db.Account{ID: 81}, "agent", "ip")
	require.ErrorIs(t, err, want)
	require.Equal(t, LoginResult{}, result, "token generation alone is not a successful session")
}
