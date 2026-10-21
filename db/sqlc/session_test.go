package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetSession(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	account, err := NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-hash",
	})
	require.NoError(t, err)
	id := uuid.New()
	token := "refresh-token-" + uuid.NewString()
	session, err := New(pool).CreateSession(ctx, CreateSessionParams{
		ID: id, AccountID: account.ID, RefreshToken: token,
		UserAgent: "test-client", ClientIp: "127.0.0.1", ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.Equal(t, id, session.ID)
	require.Equal(t, account.ID, session.AccountID)
	got, err := New(pool).GetSession(ctx, id)
	require.NoError(t, err)
	require.Equal(t, token, got.RefreshToken)
	require.False(t, got.IsBlocked)
}
