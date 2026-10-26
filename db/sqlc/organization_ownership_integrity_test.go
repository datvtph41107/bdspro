package db

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// CreateOrganizationTx must not leave a Party or Organization when the
// ownership membership fails its foreign key. The returned PartyID is
// generated inside the unsuccessful transaction, not a committed resource.
func TestCreateOrganizationTxMissingOwnerRollsBackAllEntities(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)

	org, err := store.CreateOrganizationTx(ctx, CreateOrganizationTxParams{
		Name:          "BIZ-02 invalid owner " + uuid.NewString(),
		OwnerPersonID: math.MaxInt64,
	})
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "expected a PostgreSQL foreign key rejection")
	require.Equal(t, "23503", pgErr.Code)
	require.NotZero(t, org.PartyID, "Party and Organization inserts should be attempted before FK rejection")

	var partyCount, organizationCount, membershipCount int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM parties WHERE id=$1", org.PartyID).Scan(&partyCount))
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM organizations WHERE party_id=$1", org.PartyID).Scan(&organizationCount))
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM organization_memberships WHERE organization_id=$1", org.PartyID).Scan(&membershipCount))
	require.Zero(t, partyCount, "failed creation must not leave an orphaned Party")
	require.Zero(t, organizationCount, "failed creation must not persist an Organization")
	require.Zero(t, membershipCount, "failed creation must not persist an owner membership")
}

// The PostgreSQL partial unique index, not an in-memory mutex or request
// validation, must prevent concurrent duplicate active memberships.
// This test does not model concurrent owner revocation or ownership transfers.
func TestConcurrentOrganizationMemberAdditionEnforcesOneActiveMembership(t *testing.T) {
	pool := openTestDB(t)
	ctx := context.Background()
	store := NewStore(pool)
	register := func() Account {
		t.Helper()
		account, err := store.RegisterTx(ctx, RegisterTxParams{
			Email:        uuid.NewString() + "@example.test",
			PasswordHash: "test-hash",
		})
		require.NoError(t, err)
		return account
	}
	owner := register()
	member := register()
	organization, err := store.CreateOrganizationTx(ctx, CreateOrganizationTxParams{
		Name:          "BIZ-02 concurrent membership " + uuid.NewString(),
		OwnerPersonID: owner.PersonID,
	})
	require.NoError(t, err)

	const attempts = 8
	type result struct {
		membership OrganizationMembership
		err        error
	}
	results := make(chan result, attempts)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			membership, err := store.AddOrganizationMember(ctx, AddOrganizationMemberParams{
				OrganizationID: organization.PartyID,
				PersonID:       member.PersonID,
				ActorAccountID: owner.ID,
			})
			results <- result{membership: membership, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var success, duplicate int
	for res := range results {
		if res.err == nil {
			success++
			require.Equal(t, organization.PartyID, res.membership.OrganizationID)
			require.Equal(t, member.PersonID, res.membership.PersonID)
			require.Equal(t, "member", res.membership.Role)
			continue
		}
		var pgErr *pgconn.PgError
		require.True(t, errors.As(res.err, &pgErr), "unexpected database error: %v", res.err)
		require.Equal(t, "23505", pgErr.Code, "only a uniqueness violation is expected")
		duplicate++
	}
	require.Equal(t, 1, success)
	require.Equal(t, attempts-1, duplicate)

	var activeMemberships int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*) FROM organization_memberships
		WHERE organization_id=$1 AND person_id=$2 AND ended_at IS NULL`,
		organization.PartyID, member.PersonID).Scan(&activeMemberships))
	require.Equal(t, 1, activeMemberships)
}
