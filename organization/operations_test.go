package organization

import (
	"context"
	"errors"
	"testing"

	mockdb "github.com/datvtph41107/bdspro/db/mock"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type operationContextKey struct{}

// The same HTTP request context must reach both the actor resolution and the
// PostgreSQL operation. No Gin context or HTTP response belongs in this package.
func TestCreateForAccountResolvesPersonAndPreservesContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.WithValue(context.Background(), operationContextKey{}, "request")
	expected := db.Organization{PartyID: 90, Name: "Organization A"}

	gomock.InOrder(
		store.EXPECT().GetAccountByID(ctx, int64(17)).
			Return(db.Account{ID: 17, PersonID: 42}, nil),
		store.EXPECT().CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
			OwnerPersonID: 42, Name: "Organization A",
		}).Return(expected, nil),
	)
	got, err := CreateForAccount(ctx, store, 17, "Organization A")
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestCreateForAccountStopsWhenActorResolutionFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	unavailable := errors.New("account query failed")
	store.EXPECT().GetAccountByID(ctx, int64(17)).
		Return(db.Account{}, unavailable)
	// gomock treats any unexpected transaction call as a test failure.
	organization, err := CreateForAccount(ctx, store, 17, "Name")
	require.ErrorIs(t, err, unavailable)
	require.Equal(t, db.Organization{}, organization)
}

func TestCreateForAccountPreservesTransactionFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	failure := errors.New("create organization transaction failed")
	gomock.InOrder(
		store.EXPECT().GetAccountByID(ctx, int64(17)).
			Return(db.Account{ID: 17, PersonID: 42}, nil),
		store.EXPECT().CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
			OwnerPersonID: 42, Name: "Organization A",
		}).Return(db.Organization{}, failure),
	)
	_, err := CreateForAccount(ctx, store, 17, "Organization A")
	require.ErrorIs(t, err, failure)
}

func TestListForAccountResolvesPersonAndPreservesContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.WithValue(context.Background(), operationContextKey{}, "request")
	expected := []db.Organization{{PartyID: 90, Name: "Organization A"}}
	gomock.InOrder(
		store.EXPECT().GetAccountByID(ctx, int64(17)).
			Return(db.Account{ID: 17, PersonID: 42}, nil),
		store.EXPECT().ListOrganizationsForPerson(ctx, int64(42)).
			Return(expected, nil),
	)
	got, err := ListForAccount(ctx, store, 17)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestListForAccountStopsWhenActorResolutionFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	unavailable := errors.New("account query failed")
	store.EXPECT().GetAccountByID(ctx, int64(17)).
		Return(db.Account{}, unavailable)
	organizations, err := ListForAccount(ctx, store, 17)
	require.ErrorIs(t, err, unavailable)
	require.Nil(t, organizations)
}

func TestListForAccountPreservesQueryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	failure := errors.New("list organizations failed")
	gomock.InOrder(
		store.EXPECT().GetAccountByID(ctx, int64(17)).
			Return(db.Account{ID: 17, PersonID: 42}, nil),
		store.EXPECT().ListOrganizationsForPerson(ctx, int64(42)).
			Return(nil, failure),
	)
	organizations, err := ListForAccount(ctx, store, 17)
	require.ErrorIs(t, err, failure)
	require.Nil(t, organizations)
}
