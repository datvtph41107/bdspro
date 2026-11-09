package organization

import (
	"context"
	"errors"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// GetOrgMembersForDoer resolves the verified session actor and Store from the
// request context. SQLC retains the owner/visibility predicate.
func GetOrgMembersForDoer(ctx context.Context, organizationID int64) ([]db.ListOrganizationMembersForOwnerRow, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return nil, auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return GetOrgMembers(ctx, store, actor.AccountID, organizationID)
}

func GetOrgMembers(ctx context.Context, store db.Store, actorAccountID, organizationID int64) ([]db.ListOrganizationMembersForOwnerRow, error) {
	members, err := store.ListOrganizationMembersForOwner(ctx, db.ListOrganizationMembersForOwnerParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
	})
	if err != nil {
		return nil, err
	}
	// Active owners must be active members. Preserve R01's 404 visibility policy.
	if len(members) == 0 {
		return nil, ErrOrganizationNotFound
	}
	return members, nil
}

// AddOrgUser delegates the authorization check to the single SQL INSERT WHERE
// EXISTS, while interpreting PostgreSQL constraints as model failures.
func AddOrgUser(ctx context.Context, writer OrganizationWriter, actorAccountID, organizationID, personID int64) (db.OrganizationMember, error) {
	member, err := writer.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		OrganizationID: organizationID, PersonID: personID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OrganizationMember{}, util.WithCause(ErrOrganizationOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "organization_members_one_open_uq") {
		return db.OrganizationMember{}, util.WithCause(ErrMembershipExists, err)
	}
	if isPGConstraint(err, "23503", "organization_members_person_id_fkey") {
		return db.OrganizationMember{}, util.WithCause(ErrPersonUnavailable, err)
	}
	return member, err
}

// RemoveOrdinaryOrgMemberTx ends an ordinary membership and removes all
// dependent team grants in one transaction. An owner cannot be removed by
// this operation, regardless of the number of existing owners.
func RemoveOrdinaryOrgMemberTx(ctx context.Context, q *db.Queries, actorAccountID, organizationID, personID int64) error {
	_, err := q.EndOrdinaryOrganizationMemberForOwner(ctx, db.EndOrdinaryOrganizationMemberForOwnerParams{
		OrganizationID: organizationID, PersonID: personID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrganizationMemberNotFound
	}
	if err != nil {
		return err
	}
	if _, err := q.DeleteOrgMemberTeamLinks(ctx, db.DeleteOrgMemberTeamLinksParams{
		OrganizationID: organizationID, PersonID: personID,
	}); err != nil {
		return err
	}
	return audit_model.RecordOrganizationMemberRemoved(ctx, q, actorAccountID, organizationID, personID)
}

// LockRoleChangeOrganization provides the shared serialization point for
// owner role mutations and ordinary member removal. The organization row is
// FOR NO KEY UPDATE, not FOR UPDATE: team/member inserts still need FK
// KEY SHARE of the organization and must not be trapped in a lock cycle.
func LockRoleChangeOrganization(ctx context.Context, q *db.Queries, organizationID int64) error {
	_, err := q.LockOrganizationRoleChanges(ctx, organizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrganizationNotFound
	}
	return err
}

// RequireRoleChangeOwner resolves an acting Account to an active owner Person
// in the *same transaction* that holds the organization and session locks.
// FOR SHARE prevents an owner demotion from passing a stale authorization.
func RequireRoleChangeOwner(ctx context.Context, q *db.Queries, organizationID, actorAccountID int64) (int64, error) {
	personID, err := q.LockActingOrganizationOwner(ctx, db.LockActingOrganizationOwnerParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrOrganizationOwnerRequired
	}
	return personID, err
}

// PromoteOwnerTx changes a verified, active ordinary member to an owner.
// Call only after RequireRoleChangeOwner under the same organization lock.
func PromoteOwnerTx(ctx context.Context, q *db.Queries, organizationID, personID int64) (db.OrganizationMember, error) {
	member, err := q.PromoteOrganizationMember(ctx, db.PromoteOrganizationMemberParams{
		OrganizationID: organizationID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OrganizationMember{}, ErrOwnerCandidateInvalid
	}
	return member, err
}

// DemoteOwnerTx never deletes organization membership or team grants.
// The caller must serialize owner changes and protect the last owner.
func DemoteOwnerTx(ctx context.Context, q *db.Queries, organizationID, personID int64) (db.OrganizationMember, error) {
	member, err := q.DemoteOrganizationOwner(ctx, db.DemoteOrganizationOwnerParams{
		OrganizationID: organizationID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OrganizationMember{}, ErrOwnerNotFound
	}
	return member, err
}
