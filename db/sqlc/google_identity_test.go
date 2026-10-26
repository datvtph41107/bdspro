package db

import (
	"context"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGoogleIdentityOneToOne(t *testing.T) {
	pool := openTestDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	createAccount := func() Account {
		party, err := store.CreateParty(ctx)
		require.NoError(t, err)
		require.NoError(t, store.CreatePerson(ctx, party))
		hash, err := util.HashPassword("test-password-123")
		require.NoError(t, err)
		user, err := store.CreateAccount(ctx, CreateAccountParams{
			PersonID: party, Email: uuid.NewString() + "@example.test", PasswordHash: hash,
		})
		require.NoError(t, err)
		return user
	}
	a, b := createAccount(), createAccount()
	sub1, sub2 := "google-"+uuid.NewString(), "google-"+uuid.NewString()
	require.NoError(t, store.LinkGoogleIdentity(ctx, LinkGoogleIdentityParams{AccountID: a.ID, GoogleSub: sub1}))
	require.NoError(t, store.LinkGoogleIdentity(ctx, LinkGoogleIdentityParams{AccountID: a.ID, GoogleSub: sub1}))
	linked, err := store.GetGoogleAccountID(ctx, sub1)
	require.NoError(t, err)
	require.Equal(t, a.ID, linked)
	require.Error(t, store.LinkGoogleIdentity(ctx, LinkGoogleIdentityParams{AccountID: b.ID, GoogleSub: sub1}))
	require.Error(t, store.LinkGoogleIdentity(ctx, LinkGoogleIdentityParams{AccountID: a.ID, GoogleSub: sub2}))
}
