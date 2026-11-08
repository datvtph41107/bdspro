package organization

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

type CreateTeamOptions struct {
	OrganizationID int64
	ActorAccountID int64
	Name           string
	Description    string
}

// CreateTeam checks the owner role atomically in the SQL INSERT.
func CreateTeam(ctx context.Context, store db.Store, opts CreateTeamOptions) (db.Team, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return db.Team{}, ErrTeamNameInvalid
	}
	team, err := store.CreateTeamForOwner(ctx, db.CreateTeamForOwnerParams{
		OrganizationID: opts.OrganizationID,
		ActorAccountID: opts.ActorAccountID,
		Name:           name,
		Description:    opts.Description,
	})
	return team, teamWriteError(err)
}

// GetTeam never exposes a team outside the caller's active organization membership.
func GetTeam(ctx context.Context, store db.Store, actorAccountID, teamID int64) (db.Team, error) {
	team, err := store.GetTeamForAccount(ctx, db.GetTeamForAccountParams{
		TeamID: teamID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Team{}, ErrTeamNotFound
	}
	return team, err
}

func RenameTeam(ctx context.Context, store db.Store, actorAccountID, teamID int64, name string) (db.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return db.Team{}, ErrTeamNameInvalid
	}
	team, err := store.RenameTeamForOwner(ctx, db.RenameTeamForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, Name: name,
	})
	return team, teamWriteError(err)
}

func teamWriteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return util.WithCause(ErrTeamOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "teams_org_name_uq") {
		return util.WithCause(ErrTeamAlreadyExists, err)
	}
	return err
}

// ListTeams applies the same authorization predicate as the single-team query.
func ListTeams(ctx context.Context, store db.Store, actorAccountID, organizationID int64, opts db.ListOptions) ([]db.Team, error) {
	skip, take, ok := opts.GetSkipTake32()
	if !ok {
		return nil, ErrTeamPaginationInvalid
	}
	teams, err := store.ListTeamsForAccount(ctx, db.ListTeamsForAccountParams{
		ActorAccountID: actorAccountID,
		OrganizationID: organizationID,
		Take:           take,
		Skip:           skip,
	})
	if err != nil || len(teams) != 0 {
		return teams, err
	}
	// Filtering returns no rows for both an empty page and an organization
	// invisible to this account. Resolve visibility only for empty results:
	// the normal non-empty path still uses one SQL round-trip.
	if _, err := GetForAccount(ctx, store, actorAccountID, organizationID); err != nil {
		return nil, err
	}
	return teams, nil
}
