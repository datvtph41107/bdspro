package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/val"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrForbidden = errors.New("organization operation not permitted")
	ErrInvalid   = errors.New("invalid organization request")
	ErrNotFound  = errors.New("organization request unavailable")
	ErrConflict  = errors.New("organization conflict or duplicate")
)

type Service struct{ Pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{Pool: pool} }
func requireAccount(accountID int64) error {
	if accountID <= 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Service) SubmitRegistration(ctx context.Context, accountID int64, name, jurisdiction, registration string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	name = strings.TrimSpace(name)
	jurisdiction = strings.ToUpper(strings.TrimSpace(jurisdiction))
	registration = strings.TrimSpace(registration)
	if name == "" || jurisdiction == "" || len(name) > 250 || len(jurisdiction) > 50 || len(registration) > 100 {
		return 0, ErrInvalid
	}
	id, err := dbsqlc.New(s.Pool).SubmitOrganizationRegistration(ctx, dbsqlc.SubmitOrganizationRegistrationParams{
		AccountID: accountID, Name: name, Jurisdiction: jurisdiction, Registration: registration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}

func (s *Service) SubmitClaim(ctx context.Context, accountID, organizationID int64, evidence string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	evidence = strings.TrimSpace(evidence)
	if organizationID <= 0 || evidence == "" || len(evidence) > 1000 {
		return 0, ErrInvalid
	}
	id, err := dbsqlc.New(s.Pool).SubmitOrganizationClaim(ctx, dbsqlc.SubmitOrganizationClaimParams{
		AccountID: accountID, OrganizationID: organizationID, Evidence: evidence,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}

// Only an operator (NOT tenant admin) can provision platform_organization_reviewers.
// Reviewer authority is checked and row-locked in the same transaction as approval.
func requireReviewer(ctx context.Context, q *dbsqlc.Queries, accountID int64) error {
	_, err := q.LockActivePlatformReviewer(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	return err
}

func grantInitialAdmin(ctx context.Context, q *dbsqlc.Queries, orgID, personID int64) error {
	membershipID, err := q.UpsertInitialOrganizationMembership(ctx, dbsqlc.UpsertInitialOrganizationMembershipParams{
		OrganizationID: orgID, PersonID: personID,
	})
	if err != nil {
		return err
	}
	roleID, err := q.CreateWorkspaceAdminRole(ctx, orgID)
	if err != nil {
		return err
	}
	if err := q.GrantInitialRolePermissions(ctx, roleID); err != nil {
		return err
	}
	if err := q.GrantInitialMemberRemovalPermission(ctx, roleID); err != nil {
		return err
	}
	if err := q.CreateInitialRoleAssignment(ctx, dbsqlc.CreateInitialRoleAssignmentParams{
		OrganizationID: orgID, MembershipID: membershipID, RoleID: roleID,
	}); err != nil {
		return err
	}
	return q.SetPrimaryAdministrator(ctx, dbsqlc.SetPrimaryAdministratorParams{
		OrganizationID: orgID, MembershipID: membershipID,
	})
}

// ApproveRegistration atomically creates Party+Organization, validated registry
// identity, tenant admin membership/role and a resolved review request.
func (s *Service) ApproveRegistration(ctx context.Context, reviewerID, requestID int64, reference string) (int64, error) {
	if reviewerID <= 0 || requestID <= 0 || strings.TrimSpace(reference) == "" {
		return 0, ErrInvalid
	}
	var orgID int64
	err := dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if err := requireReviewer(ctx, q, reviewerID); err != nil {
			return err
		}
		request, err := q.GetPendingOrganizationRegistrationForUpdate(ctx, requestID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if strings.TrimSpace(request.RegistrationCode) == "" {
			return fmt.Errorf("%w: verified registry identifier required", ErrInvalid)
		}
		personID, err := q.GetOpenAccountPersonID(ctx, request.RequestedByAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		orgID, err = q.CreateOrganizationParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreateOrganizationSubtype(ctx, dbsqlc.CreateOrganizationSubtypeParams{
			PartyID: orgID, Name: request.DeclaredName,
		}); err != nil {
			return err
		}
		if err := q.CreateVerifiedRegistryIdentifier(ctx, dbsqlc.CreateVerifiedRegistryIdentifierParams{
			OrganizationID: orgID, Jurisdiction: request.DeclaredJurisdictionCode,
			Registration: strings.TrimSpace(request.RegistrationCode), ReviewerID: reviewerID,
		}); err != nil {
			return err
		}
		if err := grantInitialAdmin(ctx, q, orgID, personID); err != nil {
			return err
		}
		rows, err := q.ResolveOrganizationRegistration(ctx, dbsqlc.ResolveOrganizationRegistrationParams{
			OrganizationID: pgtype.Int8{Int64: orgID, Valid: true}, ReviewerID: pgtype.Int8{Int64: reviewerID, Valid: true},
			Reference: pgtype.Text{String: strings.TrimSpace(reference), Valid: true}, RequestID: requestID,
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		if isRegistryIdentityConflict(err) {
			return 0, ErrConflict
		}
		return 0, err
	}
	return orgID, nil
}

// A claim can bootstrap the FIRST workspace admin only when no administrator
// exists. Never silently steal/replace a tenant's incumbent admin.
func (s *Service) ApproveInitialClaim(ctx context.Context, reviewerID, requestID int64, reference string) (int64, error) {
	if reviewerID <= 0 || requestID <= 0 || strings.TrimSpace(reference) == "" {
		return 0, ErrInvalid
	}
	var orgID int64
	err := dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if err := requireReviewer(ctx, q, reviewerID); err != nil {
			return err
		}
		request, err := q.GetPendingOrganizationClaimForUpdate(ctx, requestID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		orgID = request.OrganizationID
		// Organization row serializes competing initial claims of the same company.
		if _, err := q.LockActiveOrganizationForUpdate(ctx, orgID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrConflict
			}
			return err
		}
		_, err = q.GetOrganizationPrimaryAdministrator(ctx, orgID)
		if err == nil {
			return ErrConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		personID, err := q.GetOpenAccountPersonID(ctx, request.RequestedByAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		if err := grantInitialAdmin(ctx, q, orgID, personID); err != nil {
			return err
		}
		rows, err := q.ResolveOrganizationClaim(ctx, dbsqlc.ResolveOrganizationClaimParams{
			ReviewerID: pgtype.Int8{Int64: reviewerID, Valid: true}, Reference: pgtype.Text{String: strings.TrimSpace(reference), Valid: true}, RequestID: requestID,
		})
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrConflict
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return orgID, nil
}

func (s *Service) Invite(ctx context.Context, accountID, organizationID, roleID int64, email string) (int64, error) {
	if err := requireAccount(accountID); err != nil {
		return 0, err
	}
	email, err := val.Email(email)
	if organizationID <= 0 || roleID <= 0 || err != nil {
		return 0, ErrInvalid
	}
	id, err := dbsqlc.New(s.Pool).CreateAuthorizedInvitation(ctx, dbsqlc.CreateAuthorizedInvitationParams{
		AccountID: accountID, OrganizationID: organizationID, RoleID: roleID, Email: email,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrForbidden
	}
	return id, err
}

func (s *Service) AcceptInvitation(ctx context.Context, accountID, invitationID int64) (int64, error) {
	if accountID <= 0 || invitationID <= 0 {
		return 0, ErrInvalid
	}
	var membership int64
	err := dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		invitation, err := q.AcceptPendingInvitation(ctx, dbsqlc.AcceptPendingInvitationParams{
			InvitationID: invitationID, AccountID: accountID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		personID, err := q.GetInvitationAccountPersonID(ctx, accountID)
		if err != nil {
			return err
		}
		membership, err = q.CreateInvitationMembership(ctx, dbsqlc.CreateInvitationMembershipParams{
			OrganizationID: invitation.OrganizationID, PersonID: personID, InvitationID: pgtype.Int8{Int64: invitationID, Valid: true},
		})
		if err != nil {
			return err
		}
		return q.AssignInvitationRole(ctx, dbsqlc.AssignInvitationRoleParams{
			OrganizationID: invitation.OrganizationID, MembershipID: membership, RoleID: invitation.IntendedRoleID,
		})
	})
	if err != nil {
		if isInvitationMembershipConflict(err) {
			return 0, ErrConflict
		}
		return 0, err
	}
	return membership, nil
}

type Member struct {
	MembershipID int64
	PersonID     int64
	JoinedAt     time.Time
}

func (s *Service) ListMembers(ctx context.Context, accountID, organizationID int64) ([]Member, error) {
	if accountID <= 0 || organizationID <= 0 {
		return nil, ErrForbidden
	}
	rows, err := dbsqlc.New(s.Pool).ListAuthorizedMembers(ctx, dbsqlc.ListAuthorizedMembersParams{
		AccountID: accountID, OrganizationID: organizationID,
	})
	if err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(rows))
	for _, m := range rows {
		members = append(members, Member{
			MembershipID: m.MembershipID, PersonID: m.PersonID, JoinedAt: m.JoinedAt,
		})
	}
	if len(members) == 0 {
		return nil, ErrNotFound
	}
	return members, nil
}

// RemoveMember revokes a non-primary member and their active role assignments.
// An organization row lock serializes this change with Source writes. Actor
// authority is checked in SQL after acquiring the lock, never from a cookie role.
func (s *Service) RemoveMember(ctx context.Context, actorID, orgID, memberID int64) error {
	if actorID <= 0 || orgID <= 0 || memberID <= 0 {
		return ErrInvalid
	}
	return dbsqlc.NewStore(s.Pool).ExecTx(ctx, func(q *dbsqlc.Queries) error {
		if _, err := q.LockActiveOrganizationForUpdate(ctx, orgID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		n, err := q.EndAuthorizedOrganizationMembership(ctx, dbsqlc.EndAuthorizedOrganizationMembershipParams{
			OrganizationID: orgID, MembershipID: memberID, ActorAccountID: actorID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrNotFound
		}
		_, err = q.EndOrganizationMembershipRoles(ctx, dbsqlc.EndOrganizationMembershipRolesParams{
			OrganizationID: orgID, MembershipID: memberID,
		})
		return err
	})
}

// isRegistryIdentityConflict is deliberately restricted to the registry table.
// Its two UNIQUE constraints represent known registration identity conflicts.
// An unrelated UNIQUE or FK violation must surface as an internal failure,
// not masquerade as a client-caused HTTP 409.
func isRegistryIdentityConflict(err error) bool {
	pgErr := dbsqlc.PostgresError(err)
	return pgErr != nil &&
		pgErr.Code == "23505" &&
		pgErr.TableName == "organization_registry_identifiers" &&
		pgErr.ConstraintName != ""
}

// Only an already-open membership or a reused invitation is a known 409.
// New constraints must be explicitly reviewed before adding them here.
func isInvitationMembershipConflict(err error) bool {
	pgErr := dbsqlc.PostgresError(err)
	return pgErr != nil && pgErr.Code == "23505" &&
		(pgErr.ConstraintName == "organization_memberships_one_open_uq" ||
			pgErr.ConstraintName == "organization_memberships_originating_invitation_id_key")
}
