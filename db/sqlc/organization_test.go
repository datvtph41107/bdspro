package db

import (
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestOrganizationRegistrationQueries(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, applicantID := createRandomAccount(t, ctx, q)
	emailID, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: applicantID, Email: util.RandomEmail(),
	})
	require.NoError(t, err)

	args := SubmitOrganizationRegistrationParams{
		AccountID:    applicantID,
		Name:         "Sample Realty " + util.RandomString(6),
		Jurisdiction: "VN",
		Registration: "REG-" + util.RandomString(10),
	}

	// Registration requires a verified email, enforced by the query itself.
	_, err = q.SubmitOrganizationRegistration(ctx, args)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	require.NoError(t, q.MarkAccountEmailVerified(ctx, emailID))
	requestID, err := q.SubmitOrganizationRegistration(ctx, args)
	require.NoError(t, err)
	require.Positive(t, requestID)

	pending, err := q.GetPendingOrganizationRegistrationForUpdate(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, applicantID, pending.RequestedByAccountID)
	require.Equal(t, args.Name, pending.DeclaredName)
	require.Equal(t, args.Registration, pending.RegistrationCode)

	// Reviewer authority is provisioned only as a privileged test fixture.
	_, reviewerID := createRandomAccount(t, ctx, q)
	_, err = q.LockActivePlatformReviewer(ctx, reviewerID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = tx.Exec(ctx, `INSERT INTO platform_organization_reviewers
		(account_id, authorization_reference) VALUES($1,'lecture05-reviewer')`, reviewerID)
	require.NoError(t, err)
	lockedReviewer, err := q.LockActivePlatformReviewer(ctx, reviewerID)
	require.NoError(t, err)
	require.Equal(t, reviewerID, lockedReviewer)

	orgID := createRandomOrganization(t, ctx, q)
	require.NoError(t, q.CreateVerifiedRegistryIdentifier(ctx, CreateVerifiedRegistryIdentifierParams{
		OrganizationID: orgID,
		Jurisdiction:   args.Jurisdiction,
		Registration:   args.Registration,
		ReviewerID:     reviewerID,
	}))

	resolvedArgs := ResolveOrganizationRegistrationParams{
		RequestID:      requestID,
		OrganizationID: pgtype.Int8{Int64: orgID, Valid: true},
		ReviewerID:     pgtype.Int8{Int64: reviewerID, Valid: true},
		Reference:      pgtype.Text{String: "lecture05-approval", Valid: true},
	}
	rows, err := q.ResolveOrganizationRegistration(ctx, resolvedArgs)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)

	// :execrows distinguishes the first approval from a repeated decision.
	rows, err = q.ResolveOrganizationRegistration(ctx, resolvedArgs)
	require.NoError(t, err)
	require.Zero(t, rows)
}

func TestOrganizationClaimQueries(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	orgID := createRandomOrganization(t, ctx, q)
	_, actorID, _ := createVerifiedAccount(t, ctx, q)
	_, reviewerID := createRandomAccount(t, ctx, q)
	_, err := tx.Exec(ctx, `INSERT INTO platform_organization_reviewers
		(account_id, authorization_reference) VALUES($1,'lecture05-claim-reviewer')`, reviewerID)
	require.NoError(t, err)

	lockedOrg, err := q.LockActiveOrganizationForUpdate(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, orgID, lockedOrg)

	requestID, err := q.SubmitOrganizationClaim(ctx, SubmitOrganizationClaimParams{
		OrganizationID: orgID,
		AccountID:      actorID,
		Evidence:       "fixture-reference-" + util.RandomString(8),
	})
	require.NoError(t, err)
	require.Positive(t, requestID)

	pending, err := q.GetPendingOrganizationClaimForUpdate(ctx, requestID)
	require.NoError(t, err)
	require.Equal(t, orgID, pending.OrganizationID)
	require.Equal(t, actorID, pending.RequestedByAccountID)

	args := ResolveOrganizationClaimParams{
		RequestID:  requestID,
		ReviewerID: pgtype.Int8{Int64: reviewerID, Valid: true},
		Reference:  pgtype.Text{String: "verified-evidence", Valid: true},
	}
	rows, err := q.ResolveOrganizationClaim(ctx, args)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)

	rows, err = q.ResolveOrganizationClaim(ctx, args)
	require.NoError(t, err)
	require.Zero(t, rows)
}

func TestOrganizationMembershipAndInvitationQueries(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	orgID := createRandomOrganization(t, ctx, q)
	ownerPersonID, ownerAccountID, _ := createVerifiedAccount(t, ctx, q)
	inviteePersonID, inviteeAccountID, inviteeEmail := createVerifiedAccount(t, ctx, q)

	membershipID, err := q.UpsertInitialOrganizationMembership(ctx, UpsertInitialOrganizationMembershipParams{
		OrganizationID: orgID, PersonID: ownerPersonID,
	})
	require.NoError(t, err)
	require.Positive(t, membershipID)

	// Idempotent upsert returns the same still-active membership.
	again, err := q.UpsertInitialOrganizationMembership(ctx, UpsertInitialOrganizationMembershipParams{
		OrganizationID: orgID, PersonID: ownerPersonID,
	})
	require.NoError(t, err)
	require.Equal(t, membershipID, again)

	roleID, err := q.CreateWorkspaceAdminRole(ctx, orgID)
	require.NoError(t, err)
	require.Positive(t, roleID)
	require.NoError(t, q.GrantInitialRolePermissions(ctx, roleID))
	require.NoError(t, q.CreateInitialRoleAssignment(ctx, CreateInitialRoleAssignmentParams{
		OrganizationID: orgID, MembershipID: membershipID, RoleID: roleID,
	}))
	require.NoError(t, q.SetPrimaryAdministrator(ctx, SetPrimaryAdministratorParams{
		OrganizationID: orgID, MembershipID: membershipID,
	}))
	primaryID, err := q.GetOrganizationPrimaryAdministrator(ctx, orgID)
	require.NoError(t, err)
	require.Equal(t, membershipID, primaryID)

	members, err := q.ListAuthorizedMembers(ctx, ListAuthorizedMembersParams{
		OrganizationID: orgID, AccountID: ownerAccountID,
	})
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, ownerPersonID, members[0].PersonID)

	invitationID, err := q.CreateAuthorizedInvitation(ctx, CreateAuthorizedInvitationParams{
		AccountID: ownerAccountID, OrganizationID: orgID,
		RoleID: roleID, Email: inviteeEmail,
	})
	require.NoError(t, err)
	require.Positive(t, invitationID)

	// An archived workspace must not grant a new membership from an old invitation.
	_, err = tx.Exec(ctx, "UPDATE organizations SET archived_at=now() WHERE party_id=$1", orgID)
	require.NoError(t, err)
	_, err = q.AcceptPendingInvitation(ctx, AcceptPendingInvitationParams{
		InvitationID: invitationID, AccountID: inviteeAccountID,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
	_, err = tx.Exec(ctx, "UPDATE organizations SET archived_at=NULL WHERE party_id=$1", orgID)
	require.NoError(t, err)

	invitation, err := q.AcceptPendingInvitation(ctx, AcceptPendingInvitationParams{
		InvitationID: invitationID, AccountID: inviteeAccountID,
	})
	require.NoError(t, err)
	require.Equal(t, orgID, invitation.OrganizationID)
	require.Equal(t, roleID, invitation.IntendedRoleID)

	inviteeMembershipID, err := q.CreateInvitationMembership(ctx, CreateInvitationMembershipParams{
		OrganizationID: orgID, PersonID: inviteePersonID,
		InvitationID: pgtype.Int8{Int64: invitationID, Valid: true},
	})
	require.NoError(t, err)
	require.NoError(t, q.AssignInvitationRole(ctx, AssignInvitationRoleParams{
		OrganizationID: orgID, MembershipID: inviteeMembershipID, RoleID: roleID,
	}))

	members, err = q.ListAuthorizedMembers(ctx, ListAuthorizedMembersParams{
		OrganizationID: orgID, AccountID: ownerAccountID,
	})
	require.NoError(t, err)
	require.Len(t, members, 2)

	// The same invitation has been consumed, and cannot be accepted again.
	_, err = q.AcceptPendingInvitation(ctx, AcceptPendingInvitationParams{
		InvitationID: invitationID, AccountID: inviteeAccountID,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)

	_, strangerID := createRandomAccount(t, ctx, q)
	members, err = q.ListAuthorizedMembers(ctx, ListAuthorizedMembersParams{
		OrganizationID: orgID, AccountID: strangerID,
	})
	require.NoError(t, err)
	require.Empty(t, members)

	// Termination of owner's membership immediately revokes member-list authority.
	_, err = tx.Exec(ctx, `UPDATE organization_memberships SET ended_at=now() WHERE id=$1`, membershipID)
	require.NoError(t, err)
	members, err = q.ListAuthorizedMembers(ctx, ListAuthorizedMembersParams{
		OrganizationID: orgID, AccountID: ownerAccountID,
	})
	require.NoError(t, err)
	require.Empty(t, members)
}
