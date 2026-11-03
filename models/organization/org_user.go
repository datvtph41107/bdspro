package organization

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	apperrors "github.com/datvtph41107/bdspro/modules/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		return nil, apperrors.OrganizationNotFound
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
		return db.OrganizationMember{}, apperrors.OrganizationOwnerRequired
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == "organization_members_one_open_uq":
			return db.OrganizationMember{}, apperrors.Wrap(apperrors.MembershipExists, err)
		case pgErr.Code == "23503" && pgErr.ConstraintName == "organization_members_person_id_fkey":
			return db.OrganizationMember{}, apperrors.Wrap(apperrors.PersonUnavailable, err)
		}
	}
	return member, err
}

