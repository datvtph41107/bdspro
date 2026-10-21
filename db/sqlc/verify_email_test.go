package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestVerifyEmailTx(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)
	account, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "hash",
	})
	require.NoError(t, err)
	require.False(t, account.IsEmailVerified)
	row, err := store.CreateVerifyEmail(ctx, CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: "test-unique-secret",
	})
	require.NoError(t, err)
	verified, err := store.VerifyEmailTx(ctx, ConsumeVerifyEmailParams{ID: row.ID, SecretCode: row.SecretCode})
	require.NoError(t, err)
	require.True(t, verified.IsEmailVerified)
	_, err = store.VerifyEmailTx(ctx, ConsumeVerifyEmailParams{ID: row.ID, SecretCode: row.SecretCode})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
