package db

import (
	"context"
	"errors"
	"fmt"
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

	// Assert the business key, not the global count of parties shared by
	// other test packages running against the same disposable database.
	_, err = NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email: email, PasswordHash: "different",
	})
	require.Error(t, err)
	var count int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM accounts WHERE email=$1", email).Scan(&count))
	require.Equal(t, 1, count)
}

func TestRegisterTxEnqueueFailureRollsBack(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	email := uuid.NewString() + "@example.test"
	var insertedPersonID, insertedAccountID int64
	_, err := NewStore(pool).RegisterTx(ctx, RegisterTxParams{
		Email:        email,
		PasswordHash: "hash",
		AfterCreate: func(account Account) error {
			require.NotZero(t, account.ID)
			require.NotZero(t, account.PersonID)
			insertedAccountID = account.ID
			insertedPersonID = account.PersonID
			return errors.New("queue unavailable")
		},
	})
	require.EqualError(t, err, "queue unavailable")
	require.NotZero(t, insertedAccountID)
	require.NotZero(t, insertedPersonID)

	// These exact IDs were created inside the aborted transaction.
	// Other packages may insert parties concurrently; global counts are
	// not evidence that this transaction committed or rolled back.
	for _, tc := range []struct {
		query string
		id    int64
	}{
		{"SELECT count(*) FROM parties WHERE id=$1", insertedPersonID},
		{"SELECT count(*) FROM persons WHERE party_id=$1", insertedPersonID},
		{"SELECT count(*) FROM accounts WHERE id=$1", insertedAccountID},
	} {
		var count int
		require.NoError(t, pool.QueryRow(ctx, tc.query, tc.id).Scan(&count))
		require.Zero(t, count, tc.query)
	}
}

func TestGetOrCreateVerifyEmailTxReusesActiveRecord(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)
	account, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "hash",
	})
	require.NoError(t, err)

	first, err := store.GetOrCreateVerifyEmailTx(ctx, CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: "first-secret",
	})
	require.NoError(t, err)
	second, err := store.GetOrCreateVerifyEmailTx(ctx, CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: "second-secret",
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.SecretCode, second.SecretCode)
	require.Equal(t, "first-secret", second.SecretCode)

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM verify_emails WHERE account_id = $1", account.ID).Scan(&count))
	require.Equal(t, 1, count)

	_, err = pool.Exec(ctx, "UPDATE verify_emails SET expired_at = now() - interval '1 second' WHERE id = $1", first.ID)
	require.NoError(t, err)
	replacement, err := store.GetOrCreateVerifyEmailTx(ctx, CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: "replacement-secret",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.ID, replacement.ID)

	_, err = store.VerifyEmailTx(ctx, ConsumeVerifyEmailParams{ID: replacement.ID, SecretCode: replacement.SecretCode})
	require.NoError(t, err)
	_, err = store.GetOrCreateVerifyEmailTx(ctx, CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: "should-not-insert",
	})
	require.ErrorIs(t, err, ErrEmailAlreadyVerified)
}

func TestGetOrCreateVerifyEmailTxConcurrent(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)
	account, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "hash",
	})
	require.NoError(t, err)

	const workers = 5
	results := make(chan VerifyEmail, workers)
	errorsCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			row, err := store.GetOrCreateVerifyEmailTx(ctx, CreateVerifyEmailParams{
				AccountID: account.ID, Email: account.Email, SecretCode: fmt.Sprintf("secret-%d", i),
			})
			results <- row
			errorsCh <- err
		}(i)
	}
	var firstID int64
	for i := 0; i < workers; i++ {
		require.NoError(t, <-errorsCh)
		record := <-results
		if firstID == 0 {
			firstID = record.ID
		}
		require.Equal(t, firstID, record.ID)
	}
}
