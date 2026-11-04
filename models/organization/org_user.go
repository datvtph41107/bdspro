package organization

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

func GetOrgMembers(ctx context.Context, store Store, actorAccountID, organizationID int64) ([]db.ListOrganizationMembersForOwnerRow, error) {
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
func AddOrgUser(ctx context.Context, store Store, actorAccountID, organizationID, personID int64) (db.OrganizationMember, error) {
	member, err := store.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
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
