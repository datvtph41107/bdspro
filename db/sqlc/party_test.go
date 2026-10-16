package db

import (
	"context"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func createRandomOrganization(t *testing.T, ctx context.Context, q *Queries) int64 {
	t.Helper()
	orgID, err := q.CreateOrganizationParty(ctx)
	require.NoError(t, err)
	require.Positive(t, orgID)

	err = q.CreateOrganizationSubtype(ctx, CreateOrganizationSubtypeParams{
		PartyID: orgID,
		Name:    "Test Organization " + util.RandomString(10),
	})
	require.NoError(t, err)
	return orgID
}

func TestCreateOrganizationParty(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	orgID := createRandomOrganization(t, ctx, q)

	var kind, name string
	err := tx.QueryRow(ctx, `
		SELECT p.kind, o.name FROM parties p
		JOIN organizations o ON o.party_id = p.id
		WHERE p.id = $1`, orgID).Scan(&kind, &name)
	require.NoError(t, err)
	require.Equal(t, "ORGANIZATION", kind)
	require.NotEmpty(t, name)
}

func TestPartySubtypeMismatch(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	orgID, err := q.CreateOrganizationParty(ctx)
	require.NoError(t, err)

	// A PERSON subtype cannot reference an ORGANIZATION Party.
	err = q.CreatePersonSubtype(ctx, orgID)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23503", pgErr.Code) // composite FK violation
}
