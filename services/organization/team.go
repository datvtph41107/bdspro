// Package organization coordinates cross-model organization business operations.
package organization

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// TeamService combines first-party session authority with team membership
// changes. Resource permissions and audit remain owned by the SQLC model.
type TeamService struct {
	store db.Store
}

func NewTeamService(store db.Store) *TeamService { return &TeamService{store: store} }

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
