package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestOrganizationMembershipAndAuthorization(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)
	owner, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "hash",
	})
	require.NoError(t, err)
	member, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "hash",
	})
	require.NoError(t, err)
	organization, err := store.CreateOrganizationTx(ctx, CreateOrganizationTxParams{
		Name: "Example Organization", OwnerPersonID: owner.PersonID,
	})
	require.NoError(t, err)
	require.Positive(t, organization.PartyID)

	ownerOrganizations, err := store.ListOrganizationsForPerson(ctx, owner.PersonID)
	require.NoError(t, err)
	require.Len(t, ownerOrganizations, 1)
	require.Equal(t, organization.PartyID, ownerOrganizations[0].PartyID)

	joined, err := store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: organization.PartyID, PersonID: member.PersonID, ActorAccountID: owner.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "member", joined.Role)

	_, err = store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: organization.PartyID, PersonID: owner.PersonID, ActorAccountID: member.ID,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)

	memberOrganizations, err := store.ListOrganizationsForPerson(ctx, member.PersonID)
	require.NoError(t, err)
	require.Len(t, memberOrganizations, 1)
}
