package org

import (
	"context"
	"errors"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/jackc/pgx/v5"
)

type TeamMemberPage struct {
	Items []organization.TeamMembership
	Total int64
}

func toTeamMembership(r sqlc.TeamMember) organization.TeamMembership {
	return organization.TeamMembership{TeamID: decimal(r.TeamID), PersonID: decimal(r.PersonID), JoinedAt: r.JoinedAt}
}

func lockTeamOwner(ctx context.Context, actor authmodel.Actor, orgID, teamID int64) (*sqlc.Queries, error) {
	q, _, err := lockOrganization(ctx, actor, orgID, true)
	if err != nil {
		return nil, err
	}
	if _, err := GetTeam(ctx, actor.AccountID, orgID, teamID); err != nil {
		return nil, err
	}
	return q, nil
}

func ListTeamMembers(ctx context.Context, actor authmodel.Actor, orgID, teamID int64, opts db.ListOptions) (TeamMemberPage, error) {
	skip, take, ok := opts.GetSkipTake32()
	if !ok {
		return TeamMemberPage{}, errors.New("invalid pagination contract")
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (TeamMemberPage, error) {
		q, err := lockTeamOwner(txCtx, actor, orgID, teamID)
		if err != nil {
			return TeamMemberPage{}, err
		}
		total, err := q.CountTeamUsersForOwner(txCtx, sqlc.CountTeamUsersForOwnerParams{TeamID: teamID, ActorAccountID: actor.AccountID})
		if err != nil {
			return TeamMemberPage{}, err
		}
		rows, err := q.ListTeamUsersForAccount(txCtx, sqlc.ListTeamUsersForAccountParams{TeamID: teamID, ActorAccountID: actor.AccountID, Skip: skip, Take: take})
		if err != nil {
			return TeamMemberPage{}, err
		}
		result := TeamMemberPage{Items: make([]organization.TeamMembership, 0, len(rows)), Total: total}
		for _, row := range rows {
			result.Items = append(result.Items, toTeamMembership(row))
		}
		return result, nil
	})
}

func GetOwnTeamMembership(ctx context.Context, actorID, orgID, teamID int64) (organization.TeamMembership, error) {
	if _, err := GetTeam(ctx, actorID, orgID, teamID); err != nil {
		return organization.TeamMembership{}, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return organization.TeamMembership{}, err
	}
	row, err := q.GetOwnTeamMembership(ctx, sqlc.GetOwnTeamMembershipParams{TeamID: teamID, AccountID: actorID})
	if errors.Is(err, pgx.ErrNoRows) {
		return organization.TeamMembership{}, organization.ErrTeamMemberNotFound
	}
	if err != nil {
		return organization.TeamMembership{}, err
	}
	return toTeamMembership(row), nil
}

func teamAudit(ctx context.Context, q *sqlc.Queries, actorID, teamID int64, action, targetType, targetRef string) error {
	return q.InsertAuditAction(ctx, sqlc.InsertAuditActionParams{
		Action: action, ActorAccountID: actorID, ScopeType: "team", ScopeRef: decimal(teamID),
		TargetType: targetType, TargetRef: targetRef,
	})
}

func AddTeamMember(ctx context.Context, actor authmodel.Actor, orgID, teamID, personID int64) (organization.TeamMembership, error) {
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.TeamMembership, error) {
		q, err := lockTeamOwner(txCtx, actor, orgID, teamID)
		if err != nil {
			return organization.TeamMembership{}, err
		}
		row, err := q.AddTeamUserForOwner(txCtx, sqlc.AddTeamUserForOwnerParams{TeamID: teamID, ActorAccountID: actor.AccountID, PersonID: personID})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.TeamMembership{}, organization.ErrTeamMemberNotFound
		}
		if db.IsUniqueViolation(err, "team_members_pkey") {
			return organization.TeamMembership{}, organization.ErrTeamMemberExists
		}
		if err != nil {
			return organization.TeamMembership{}, err
		}
		if err := teamAudit(txCtx, q, actor.AccountID, teamID, "team.member.added", "person", decimal(personID)); err != nil {
			return organization.TeamMembership{}, err
		}
		return toTeamMembership(row), nil
	})
}

func RemoveTeamMember(ctx context.Context, actor authmodel.Actor, orgID, teamID, personID int64) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := lockTeamOwner(txCtx, actor, orgID, teamID)
		if err != nil {
			return err
		}
		n, err := q.RemoveTeamUserForOwner(txCtx, sqlc.RemoveTeamUserForOwnerParams{TeamID: teamID, ActorAccountID: actor.AccountID, PersonID: personID})
		if err != nil {
			return err
		}
		if n != 1 {
			return organization.ErrTeamMemberNotFound
		}
		return teamAudit(txCtx, q, actor.AccountID, teamID, "team.member.removed", "person", decimal(personID))
	})
}
