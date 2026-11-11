package organization

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// Team is the model passed from HTTP action through service to persistence.
type Team struct {
	ID          int64
	OrgID       int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func TeamFromDB(row db.Team) *Team {
	return &Team{
		ID:          row.ID,
		OrgID:       row.OrganizationID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// InsertTeam requires an existing SQLC transaction-scoped executor.
func InsertTeam(ctx context.Context, actorAccountID int64, input *Team) (*Team, error) {
	if input == nil {
		return nil, ErrTeamNameInvalid
	}
	q, err := db.TxQueries(ctx)
	if err != nil {
		return nil, err
	}
	row, err := CreateTeam(ctx, q, CreateTeamOptions{
		OrganizationID: input.OrgID,
		ActorAccountID: actorAccountID,
		Name:           input.Name,
		Description:    input.Description,
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

type CreateTeamOptions struct {
	OrganizationID int64
	ActorAccountID int64
	Name           string
	Description    string
}

// TeamWriter is the two SQLC mutations shared by direct model callers and
// transaction-bound service calls. It is not a new transaction abstraction.
type TeamWriter interface {
	CreateTeamForOwner(context.Context, db.CreateTeamForOwnerParams) (db.Team, error)
	RenameTeamForOwner(context.Context, db.RenameTeamForOwnerParams) (db.Team, error)
}

// ValidTeamName is a DB-free preflight check shared by HTTP service
// orchestration and the standalone model entry point.
func ValidTeamName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && utf8.RuneCountInString(name) <= 100
}

// CreateTeam checks the owner role atomically in the SQL INSERT.
func CreateTeam(ctx context.Context, writer TeamWriter, opts CreateTeamOptions) (*Team, error) {
	name := strings.TrimSpace(opts.Name)
	if !ValidTeamName(name) {
		return nil, ErrTeamNameInvalid
	}
	team, err := writer.CreateTeamForOwner(ctx, db.CreateTeamForOwnerParams{
		OrganizationID: opts.OrganizationID,
		ActorAccountID: opts.ActorAccountID,
		Name:           name,
		Description:    opts.Description,
	})
	if err := teamWriteError(err); err != nil {
		return nil, err
	}
	return TeamFromDB(team), nil
}

// UpdateTeam resolves the SQLC writer from the enclosing transaction context.
// It mirrors Gitea's domain Team mutation: service callers pass a model, not
// a Store or generated SQLC update parameters.
func UpdateTeam(ctx context.Context, actorAccountID int64, team *Team) (*Team, error) {
	if team == nil || team.ID <= 0 {
		return nil, ErrTeamNotFound
	}
	q, err := db.TxQueries(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := RenameTeam(ctx, q, actorAccountID, team.ID, team.Name)
	if err != nil {
		return nil, err
	}
	return saved, nil
}

// GetTeam never exposes a team outside the caller's active organization membership.
func GetTeam(ctx context.Context, store db.Querier, actorAccountID, teamID int64) (*Team, error) {
	row, err := store.GetTeamForAccount(ctx, db.GetTeamForAccountParams{
		TeamID: teamID, ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTeamNotFound
	}
	if err != nil {
		return nil, err
	}
	return TeamFromDB(row), nil
}

func RenameTeam(ctx context.Context, writer TeamWriter, actorAccountID, teamID int64, name string) (*Team, error) {
	name = strings.TrimSpace(name)
	if !ValidTeamName(name) {
		return nil, ErrTeamNameInvalid
	}
	team, err := writer.RenameTeamForOwner(ctx, db.RenameTeamForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, Name: name,
	})
	if err := teamWriteError(err); err != nil {
		return nil, err
	}
	return TeamFromDB(team), nil
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

// ListTeamsForDoer uses the verified Account actor and the Store installed
// in context by the application entrypoint. The account-scoped SQL predicate
// remains in ListTeams; no public HTTP context or authorization snapshot is
// accepted here.
func ListTeamsForDoer(ctx context.Context, organizationID int64, opts db.ListOptions) ([]*Team, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return nil, auth_model.ErrSessionUnavailable
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	return ListTeams(ctx, q, actor.AccountID, organizationID, opts)
}

// ListTeams applies the same authorization predicate as the single-team query.
func ListTeams(ctx context.Context, store db.Querier, actorAccountID, organizationID int64, opts db.ListOptions) ([]*Team, error) {
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
	if err != nil {
		return nil, err
	}
	// Filtering returns no rows for both an empty page and an organization
	// invisible to this account. Resolve visibility only for empty results:
	// the normal non-empty path still uses one SQL round-trip.
	if len(teams) == 0 {
		if _, err := GetForAccount(ctx, store, actorAccountID, organizationID); err != nil {
			return nil, err
		}
	}
	return teamModelsFromDB(teams), nil
}

// teamModelsFromDB is the one list projection into domain Team values.
// Preserve nil vs an allocated empty list for the existing JSON contract.
func teamModelsFromDB(rows []db.Team) []*Team {
	if rows == nil {
		return nil
	}
	teams := make([]*Team, 0, len(rows))
	for _, row := range rows {
		teams = append(teams, TeamFromDB(row))
	}
	return teams
}
