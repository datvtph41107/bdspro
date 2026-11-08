// Package organization coordinates cross-model organization business operations.
package organization

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// TeamService coordinates acting-session authority with team lifecycle and
// membership changes. Resource permissions and audit remain owned by SQLC.
type TeamService struct {
	store db.Store
}

func NewTeamService(store db.Store) *TeamService { return &TeamService{store: store} }

// CreateTeam is a cross-model write: a live session and the SQLC owner
// membership predicate must stay valid in the same committing transaction.
// Validate name before opening a transaction; the model validates it again
// when called directly without a service.
func (s *TeamService) CreateTeam(ctx context.Context, actingSessionID uuid.UUID, opts organization_model.CreateTeamOptions) (db.Team, error) {
	if !organization_model.ValidTeamName(opts.Name) {
		return db.Team{}, organization_model.ErrTeamNameInvalid
	}
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.Team, error) {
		if err := auth_model.RequireActingSession(ctx, q, opts.ActorAccountID, actingSessionID); err != nil {
			return db.Team{}, err
		}
		return organization_model.CreateTeam(ctx, q, opts)
	})
}

// RenameTeam preserves the owner-scoped SQLC UPDATE and its permission lock;
// the acting session row is locked first in the same transaction.
func (s *TeamService) RenameTeam(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID int64, name string) (db.Team, error) {
	if !organization_model.ValidTeamName(name) {
		return db.Team{}, organization_model.ErrTeamNameInvalid
	}
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.Team, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return db.Team{}, err
		}
		return organization_model.RenameTeam(ctx, q, actorAccountID, teamID, name)
	})
}

func (s *TeamService) AddTeamUser(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID, personID int64) (db.TeamMember, error) {
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.TeamMember, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return db.TeamMember{}, err
		}
		// Owner and recipient membership checks stay inside the SQLC
		// INSERT with FOR SHARE OF m, not in a separate preflight SELECT.
		return organization_model.AddTeamUser(ctx, q, actorAccountID, teamID, personID)
	})
}

func (s *TeamService) RemoveTeamUser(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID, personID int64) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return err
		}
		// DELETE, audit event and authority check must commit together.
		return organization_model.RemoveTeamUserTx(ctx, q, actorAccountID, teamID, personID)
	})
}
