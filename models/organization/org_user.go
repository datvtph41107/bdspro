package organization

import (
	"context"
	"errors"
	"time"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// Member is the Organization membership domain model. SQLC's nullable
// fields and owner foreign-key projection never leave models/organization.
// Like Gitea's models/organization.OrgUser, it is owned by the domain package.
type Member struct {
	ID             int64
	OrganizationID int64
	PersonID       int64
	Role           string
	JoinedAt       time.Time
	EndedAt        *time.Time
}

// MemberFromDB is the canonical SQLC membership to domain mapping.
func MemberFromDB(row db.OrganizationMember) *Member {
	member := &Member{
		ID: row.ID, OrganizationID: row.OrganizationID,
		PersonID: row.PersonID, Role: row.Role, JoinedAt: row.JoinedAt,
	}
	if row.EndedAt.Valid {
		endedAt := row.EndedAt.Time
		member.EndedAt = &endedAt
	}
	return member
}

// memberFromListRow owns the owner-scoped SQLC read projection.
func memberFromListRow(row db.ListOrganizationMembersForOwnerRow) *Member {
	return &Member{
		ID: row.ID, OrganizationID: row.OrganizationID,
		PersonID: row.PersonID, Role: row.Role, JoinedAt: row.JoinedAt,
	}
}

// GetOrgMembersForDoer resolves the verified session actor and Store from the
// request context. SQLC retains the owner/visibility predicate.
func GetOrgMembersForDoer(ctx context.Context, organizationID int64) ([]*Member, error) {
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

func GetOrgMembers(ctx context.Context, store db.Store, actorAccountID, organizationID int64) ([]*Member, error) {
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
	out := make([]*Member, 0, len(members))
	for _, row := range members {
		out = append(out, memberFromListRow(row))
	}
	return out, nil
}

// AddOrgUser delegates the authorization check to the single SQL INSERT WHERE
// EXISTS, while interpreting PostgreSQL constraints as model failures.
func AddOrgUser(ctx context.Context, writer OrganizationWriter, actorAccountID, organizationID, personID int64) (*Member, error) {
	member, err := writer.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		OrganizationID: organizationID, PersonID: personID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, util.WithCause(ErrOrganizationOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "organization_members_one_open_uq") {
		return nil, util.WithCause(ErrMembershipExists, err)
	}
	if isPGConstraint(err, "23503", "organization_members_person_id_fkey") {
		return nil, util.WithCause(ErrPersonUnavailable, err)
	}
	if err != nil {
		return nil, err
	}
	return MemberFromDB(member), nil
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
func PromoteOwnerTx(ctx context.Context, q *db.Queries, organizationID, personID int64) (*Member, error) {
	member, err := q.PromoteOrganizationMember(ctx, db.PromoteOrganizationMemberParams{
		OrganizationID: organizationID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOwnerCandidateInvalid
	}
	if err != nil {
		return nil, err
	}
	return MemberFromDB(member), nil
}

// DemoteOwnerTx never deletes organization membership or team grants.
// The caller must serialize owner changes and protect the last owner.
func DemoteOwnerTx(ctx context.Context, q *db.Queries, organizationID, personID int64) (*Member, error) {
	member, err := q.DemoteOrganizationOwner(ctx, db.DemoteOrganizationOwnerParams{
		OrganizationID: organizationID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOwnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return MemberFromDB(member), nil
}
