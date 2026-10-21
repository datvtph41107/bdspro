package db

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterTx(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	email := uuid.NewString() + "@example.test"
	account, err := NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email: email, PasswordHash: "test-hash",
	})
	require.NoError(t, err)
	require.NotZero(t, account.ID)
	require.NotZero(t, account.PersonID)

	var personExists bool
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM persons WHERE party_id=$1)",
		account.PersonID).Scan(&personExists))
	require.True(t, personExists)

	var partiesBefore int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&partiesBefore))
	_, err = NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email: email, PasswordHash: "different",
	})
	require.Error(t, err)
	var partiesAfter, count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&partiesAfter))
	require.Equal(t, partiesBefore, partiesAfter, "duplicate registration must roll back the inserted Party")
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM accounts WHERE email=$1", email).Scan(&count))
	require.Equal(t, 1, count)
}

func TestRegisterTxEnqueueFailureRollsBack(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	email := uuid.NewString() + "@example.test"
	var before int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&before))

	_, err := NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email:        email,
		PasswordHash: "hash",
		AfterCreate: func(account Account) error {
			require.NotZero(t, account.ID)
			return errors.New("queue unavailable")
		},
	})
	require.EqualError(t, err, "queue unavailable")

	var after int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after))
	require.Equal(t, before, after)
}
