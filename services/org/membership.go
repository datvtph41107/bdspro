package org

import (
	"context"
	"errors"
	"strconv"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/datvtph41107/bdspro/models/organization"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func decimal(id int64) string { return strconv.FormatInt(id, 10) }

func detail(id int64, name string, created time.Time, membership int64, role string, joined time.Time) organization.Detail {
	return organization.Detail{ID: decimal(id), Name: name, CreatedAt: created, MembershipID: decimal(membership), Role: role, JoinedAt: joined}
}

func member(id, orgID, personID int64, role string, joined time.Time) organization.Membership {
	return organization.Membership{ID: decimal(id), OrganizationID: decimal(orgID), PersonID: decimal(personID), Role: role, JoinedAt: joined}
}

func ListOrganizations(ctx context.Context, accountID int64) ([]organization.Detail, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListOrganizationsForAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	result := make([]organization.Detail, 0, len(rows))
	for _, r := range rows {
		result = append(result, detail(r.PartyID, r.Name, r.CreatedAt, r.MembershipID, r.Role, r.JoinedAt))
	}
	return result, nil
}

func GetOrganizationDetail(ctx context.Context, accountID, orgID int64) (organization.Detail, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return organization.Detail{}, err
	}
	r, err := q.GetOrganizationForAccount(ctx, sqlc.GetOrganizationForAccountParams{AccountID: accountID, OrganizationID: orgID})
	if errors.Is(err, pgx.ErrNoRows) {
		return organization.Detail{}, organization.ErrOrganizationNotFound
	}
	if err != nil {
		return organization.Detail{}, err
	}
	return detail(r.PartyID, r.Name, r.CreatedAt, r.MembershipID, r.Role, r.JoinedAt), nil
}

func CreateOrganization(ctx context.Context, actor authmodel.Actor, raw string) (organization.Detail, error) {
	name, err := organization.NormalizeOrganizationName(raw)
	if err != nil {
		return organization.Detail{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.Detail, error) {
		q, err := authservice.LockActorForMutation(txCtx, actor)
		if err != nil {
			return organization.Detail{}, err
		}
		u, err := authservice.GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return organization.Detail{}, err
		}
		if !u.Credentials.EmailVerified && !u.Credentials.PhoneVerified {
			return organization.Detail{}, organization.ErrVerifiedContactRequired
		}
		personID, err := q.LockOrganizationCreator(txCtx, actor.AccountID)
		if err != nil {
			return organization.Detail{}, err
		}
		id, err := q.CreateParty(txCtx)
		if err != nil {
			return organization.Detail{}, err
		}
		r, err := q.CreateOrganization(txCtx, sqlc.CreateOrganizationParams{PartyID: id, Name: name})
		if err != nil {
			return organization.Detail{}, err
		}
		m, err := q.CreateOrganizationMember(txCtx, sqlc.CreateOrganizationMemberParams{OrganizationID: id, PersonID: personID, Role: "owner"})
		if err != nil {
			return organization.Detail{}, err
		}
		return detail(r.PartyID, r.Name, r.CreatedAt, m.ID, m.Role, m.JoinedAt), nil
	})
}

func lockOrganization(ctx context.Context, actor authmodel.Actor, orgID int64, owner bool) (*sqlc.Queries, int64, error) {
	q, err := authservice.LockActorForMutation(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	if _, err := q.LockOrganizationRoleChanges(ctx, orgID); errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, organization.ErrOrganizationNotFound
	} else if err != nil {
		return nil, 0, err
	}
	if !owner {
		return q, 0, nil
	}
	person, err := q.LockActingOrganizationOwner(ctx, sqlc.LockActingOrganizationOwnerParams{OrganizationID: orgID, ActorAccountID: actor.AccountID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, organization.ErrOrganizationForbidden
	}
	return q, person, err
}

func UpdateOrganization(ctx context.Context, actor authmodel.Actor, orgID int64, raw string) (organization.Detail, error) {
	name, err := organization.NormalizeOrganizationName(raw)
	if err != nil {
		return organization.Detail{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.Detail, error) {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return organization.Detail{}, err
		}
		_, err = q.UpdateOrganization(txCtx, sqlc.UpdateOrganizationParams{OrganizationID: orgID, ActorAccountID: actor.AccountID, Name: pgtype.Text{String: name, Valid: true}})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Detail{}, organization.ErrOrganizationForbidden
		}
		if err != nil {
			return organization.Detail{}, err
		}
		return GetOrganizationDetail(txCtx, actor.AccountID, orgID)
	})
}

func ListOrganizationMembers(ctx context.Context, actor authmodel.Actor, orgID int64) ([]organization.Membership, error) {
	return db.WithTx2(ctx, func(txCtx context.Context) ([]organization.Membership, error) {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return nil, err
		}
		rows, err := q.ListOrganizationMembersForOwner(txCtx, sqlc.ListOrganizationMembersForOwnerParams{OrganizationID: orgID, ActorAccountID: actor.AccountID})
		if err != nil {
			return nil, err
		}
		result := make([]organization.Membership, 0, len(rows))
		for _, r := range rows {
			result = append(result, member(r.ID, r.OrganizationID, r.PersonID, r.Role, r.JoinedAt))
		}
		return result, nil
	})
}

func AddOrganizationMember(ctx context.Context, actor authmodel.Actor, orgID, personID int64) (organization.Membership, error) {
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (organization.Membership, error) {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return organization.Membership{}, err
		}
		if _, err := q.LockOrganizationMemberAccount(txCtx, personID); errors.Is(err, pgx.ErrNoRows) {
			return organization.Membership{}, organization.ErrMemberNotFound
		} else if err != nil {
			return organization.Membership{}, err
		}
		r, err := q.AddOrganizationMember(txCtx, sqlc.AddOrganizationMemberParams{OrganizationID: orgID, ActorAccountID: actor.AccountID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Membership{}, organization.ErrMemberNotFound
		}
		if err != nil {
			return organization.Membership{}, err
		}
		return member(r.ID, r.OrganizationID, r.PersonID, r.Role, r.JoinedAt), nil
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "organization_members_one_open_uq" {
		err = organization.ErrMemberExists
	}
	return result, err
}

func organizationAudit(ctx context.Context, q *sqlc.Queries, actorID, orgID, personID int64, action string) error {
	return q.InsertAuditAction(ctx, sqlc.InsertAuditActionParams{
		Action: action, ActorAccountID: actorID, ScopeType: "organization", ScopeRef: decimal(orgID),
		TargetType: "person", TargetRef: decimal(personID),
	})
}

func endTeamAuthority(ctx context.Context, q *sqlc.Queries, orgID, personID int64) error {
	_, err := q.DeleteOrgMemberTeamLinks(ctx, sqlc.DeleteOrgMemberTeamLinksParams{OrganizationID: orgID, PersonID: personID})
	return err
}

func RemoveOrganizationMember(ctx context.Context, actor authmodel.Actor, orgID, personID int64) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return err
		}
		_, err = q.EndOrdinaryOrganizationMemberForOwner(txCtx, sqlc.EndOrdinaryOrganizationMemberForOwnerParams{OrganizationID: orgID, ActorAccountID: actor.AccountID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.ErrMemberNotFound
		}
		if err != nil {
			return err
		}
		if err := endTeamAuthority(txCtx, q, orgID, personID); err != nil {
			return err
		}
		return organizationAudit(txCtx, q, actor.AccountID, orgID, personID, "organization.member.removed")
	})
}

func ensureEligibleOwner(ctx context.Context, q *sqlc.Queries, orgID int64) error {
	n, err := q.CountEligibleOrganizationOwners(ctx, orgID)
	if err != nil {
		return err
	}
	if n < 1 {
		return organization.ErrLastOwner
	}
	return nil
}

func LeaveOrganization(ctx context.Context, actor authmodel.Actor, orgID int64) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, _, err := lockOrganization(txCtx, actor, orgID, false)
		if err != nil {
			return err
		}
		r, err := q.LockOwnOrganizationMembership(txCtx, sqlc.LockOwnOrganizationMembershipParams{OrganizationID: orgID, AccountID: actor.AccountID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.ErrOrganizationNotFound
		}
		if err != nil {
			return err
		}
		n, err := q.EndOwnOrganizationMembership(txCtx, sqlc.EndOwnOrganizationMembershipParams{OrganizationID: orgID, PersonID: r.PersonID})
		if err != nil {
			return err
		}
		if n != 1 {
			return organization.ErrOrganizationNotFound
		}
		if r.Role == "owner" {
			if err := ensureEligibleOwner(txCtx, q, orgID); err != nil {
				return err
			}
		}
		if err := endTeamAuthority(txCtx, q, orgID, r.PersonID); err != nil {
			return err
		}
		return organizationAudit(txCtx, q, actor.AccountID, orgID, r.PersonID, "organization.member.removed")
	})
}

func PromoteOrganizationMember(ctx context.Context, actor authmodel.Actor, orgID, personID int64) (organization.Membership, error) {
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.Membership, error) {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return organization.Membership{}, err
		}
		r, err := q.PromoteOrganizationMember(txCtx, sqlc.PromoteOrganizationMemberParams{OrganizationID: orgID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Membership{}, organization.ErrMemberNotFound
		}
		if err != nil {
			return organization.Membership{}, err
		}
		if err := organizationAudit(txCtx, q, actor.AccountID, orgID, personID, "organization.owner.promoted"); err != nil {
			return organization.Membership{}, err
		}
		return member(r.ID, r.OrganizationID, r.PersonID, r.Role, r.JoinedAt), nil
	})
}

func DemoteOrganizationOwner(ctx context.Context, actor authmodel.Actor, orgID, personID int64) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, _, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return err
		}
		_, err = q.DemoteOrganizationOwner(txCtx, sqlc.DemoteOrganizationOwnerParams{OrganizationID: orgID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.ErrMemberNotFound
		}
		if err != nil {
			return err
		}
		if err := ensureEligibleOwner(txCtx, q, orgID); err != nil {
			return err
		}
		return organizationAudit(txCtx, q, actor.AccountID, orgID, personID, "organization.owner.demoted")
	})
}

func TransferOrganizationOwner(ctx context.Context, actor authmodel.Actor, orgID, personID int64, proof string) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, donor, err := lockOrganization(txCtx, actor, orgID, true)
		if err != nil {
			return err
		}
		if donor == personID {
			return organization.ErrInvalidOwnerTarget
		}
		if err := authservice.ConsumeRecentAuthProof(txCtx, q, actor, proof, "organization.owner.transfer"); err != nil {
			return err
		}
		_, err = q.PromoteOrganizationMember(txCtx, sqlc.PromoteOrganizationMemberParams{OrganizationID: orgID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.ErrInvalidOwnerTarget
		}
		if err != nil {
			return err
		}
		if _, err := q.DemoteOrganizationOwner(txCtx, sqlc.DemoteOrganizationOwnerParams{OrganizationID: orgID, PersonID: donor}); err != nil {
			return err
		}
		if err := ensureEligibleOwner(txCtx, q, orgID); err != nil {
			return err
		}
		return organizationAudit(txCtx, q, actor.AccountID, orgID, personID, "organization.owner.transferred")
	})
}
