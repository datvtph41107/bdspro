package organization

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	db "github.com/datvtph41107/bdspro/models/db"
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
