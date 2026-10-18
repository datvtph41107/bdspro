package organization

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestRegistryConflictRequiresKnownTable(t *testing.T) {
	registryUnique := &pgconn.PgError{
		Code: "23505", TableName: "organization_registry_identifiers",
		ConstraintName: "organization_registry_identifiers_organization_id_key",
	}
	require.True(t, isRegistryIdentityConflict(registryUnique))
	require.True(t, isRegistryIdentityConflict(fmt.Errorf("transaction: %w", registryUnique)))
	require.False(t, isRegistryIdentityConflict(&pgconn.PgError{
		Code: "23505", TableName: "organization_roles",
		ConstraintName: "organization_roles_organization_id_name_key",
	}))
	require.False(t, isRegistryIdentityConflict(&pgconn.PgError{
		Code: "23503", TableName: "organization_registry_identifiers",
		ConstraintName: "organization_registry_identifiers_organization_id_fkey",
	}))
	require.False(t, isRegistryIdentityConflict(errors.New("connection reset")))
	require.False(t, isRegistryIdentityConflict(nil))
}

func TestInvitationConflictRequiresSpecificConstraint(t *testing.T) {
	for _, name := range []string{
		"organization_memberships_one_open_uq",
		"organization_memberships_originating_invitation_id_key",
	} {
		err := &pgconn.PgError{Code: "23505", ConstraintName: name}
		require.True(t, isInvitationMembershipConflict(fmt.Errorf("invite: %w", err)))
	}
	for _, err := range []error{
		&pgconn.PgError{Code: "23505", ConstraintName: "organization_roles_organization_id_name_key"},
		&pgconn.PgError{Code: "23503", ConstraintName: "organization_memberships_one_open_uq"},
		&pgconn.PgError{Code: "40P01", ConstraintName: "organization_memberships_one_open_uq"},
		context.Canceled,
		nil,
	} {
		require.False(t, isInvitationMembershipConflict(err))
	}
}
