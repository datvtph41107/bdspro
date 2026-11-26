package org

import (
	"context"
	"errors"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func toTeam(row db.Team) organization.Team {
	return organization.Team{
		ID: row.ID, OrganizationID: row.OrganizationID,
		Name: row.Name, Description: row.Description,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type TeamPage struct {
	Items []organization.Team
	Total int64
}

func ListTeams(ctx context.Context, actorID, orgID int64, opts db.ListOptions) (TeamPage, error) {
	offset, limit, ok := opts.GetSkipTake32()
	if !ok {
		return TeamPage{}, errors.New("invalid pagination contract")
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return TeamPage{}, err
	}
	count, err := q.CountTeamsForAccount(ctx, db.CountTeamsForAccountParams{
		OrganizationID: orgID, ActorAccountID: actorID,
	})
	if err != nil {
		return TeamPage{}, err
	}
	items, err := q.ListTeamsForAccount(ctx, db.ListTeamsForAccountParams{
		ActorAccountID: actorID, OrganizationID: orgID,
		Skip: offset, Take: limit,
	})
	if err != nil {
		return TeamPage{}, err
	}
	result := TeamPage{Items: make([]organization.Team, 0, len(items)), Total: count}
	for _, item := range items {
		result.Items = append(result.Items, toTeam(item))
	}
	return result, nil
}

func GetTeam(ctx context.Context, actorID, orgID, teamID int64) (organization.Team, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return organization.Team{}, err
	}
	row, err := q.GetTeamForAccount(ctx, db.GetTeamForAccountParams{
		ActorAccountID: actorID, TeamID: teamID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return organization.Team{}, organization.ErrTeamNotFound
	}
	if err != nil {
		return organization.Team{}, err
	}
	if row.OrganizationID != orgID {
		return organization.Team{}, organization.ErrTeamNotFound
	}
	return toTeam(row), nil
}

func CreateTeam(ctx context.Context, actorID, orgID int64, name, description string) (organization.Team, error) {
	name, err := organization.NormalizeTeamName(name)
	if err != nil {
		return organization.Team{}, err
	}
	if len(description) > 2000 {
		return organization.Team{}, organization.ErrTeamDescription
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.Team, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return organization.Team{}, err
		}
		row, err := q.CreateTeamForOwner(txCtx, db.CreateTeamForOwnerParams{
			ActorAccountID: actorID, OrganizationID: orgID,
			Name: name, Description: description,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Team{}, organization.ErrTeamForbidden
		}
		if err != nil {
			return organization.Team{}, mapTeamWriteError(err)
		}
		return toTeam(row), nil
	})
}

func RenameTeam(ctx context.Context, actorID, orgID, teamID int64, name string) (organization.Team, error) {
	name, err := organization.NormalizeTeamName(name)
	if err != nil {
		return organization.Team{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (organization.Team, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return organization.Team{}, err
		}
		current, err := q.GetTeamForAccount(txCtx, db.GetTeamForAccountParams{
			ActorAccountID: actorID, TeamID: teamID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Team{}, organization.ErrTeamNotFound
		}
		if err != nil {
			return organization.Team{}, err
		}
		if current.OrganizationID != orgID {
			return organization.Team{}, organization.ErrTeamNotFound
		}
		row, err := q.RenameTeamForOwner(txCtx, db.RenameTeamForOwnerParams{
			ActorAccountID: actorID, TeamID: teamID, Name: name,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.Team{}, organization.ErrTeamForbidden
		}
		if err != nil {
			return organization.Team{}, mapTeamWriteError(err)
		}
		return toTeam(row), nil
	})
}

func mapTeamWriteError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return organization.ErrTeamExists
	}
	return err
}
