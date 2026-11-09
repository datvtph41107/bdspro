// Package org coordinates cross-model organization business operations.
package org

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// NewTeam coordinates verified session authority and Team creation in one TX.
// The original SQLC owner predicate remains the committing authorization check.
func NewTeam(ctx context.Context, team *organization_model.Team) error {
	if team == nil || !organization_model.ValidTeamName(team.Name) {
		return organization_model.ErrTeamNameInvalid
	}
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	saved, err := db.WithTxValue(ctx, func(txCtx context.Context) (*organization_model.Team, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		return organization_model.InsertTeam(txCtx, actor.AccountID, team)
	})
	if err != nil {
		return err
	}
	// Do not show a persisted in-memory model when COMMIT failed.
	*team = *saved
	return nil
}

// AddTeamMember owns current-session authorization and committing membership.
func AddTeamMember(ctx context.Context, teamID, personID int64) (db.TeamMember, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return db.TeamMember{}, auth_model.ErrSessionUnavailable
	}
	return db.WithTxValue(ctx, func(txCtx context.Context) (db.TeamMember, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return db.TeamMember{}, err
		}
		return addTeamMemberTx(txCtx, q, actor.AccountID, actor.SessionID, teamID, personID)
	})
}

// RemoveTeamMember includes session lock, SQLC owner predicate, delete and audit.
func RemoveTeamMember(ctx context.Context, teamID, personID int64) error {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return err
		}
		return removeTeamMemberTx(txCtx, q, actor.AccountID, actor.SessionID, teamID, personID)
	})
}

// Both the receiver and the standalone function use one SQLC operation.
func addTeamMemberTx(ctx context.Context, q *db.Queries, accountID int64, sessionID uuid.UUID, teamID, personID int64) (db.TeamMember, error) {
	if err := auth_model.RequireActingSession(ctx, q, accountID, sessionID); err != nil {
		return db.TeamMember{}, err
	}
	return organization_model.AddTeamUser(ctx, q, accountID, teamID, personID)
}

func removeTeamMemberTx(ctx context.Context, q *db.Queries, accountID int64, sessionID uuid.UUID, teamID, personID int64) error {
	if err := auth_model.RequireActingSession(ctx, q, accountID, sessionID); err != nil {
		return err
	}
	return organization_model.RemoveTeamUserTx(ctx, q, accountID, teamID, personID)
}

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

// RenameTeam is the request-context API used by the typed Team endpoint.
// The live acting principal and DB capability come from middleware; the
// TeamService method remains the transactional implementation shared by tests.
func RenameTeam(ctx context.Context, teamID int64, name string) (db.Team, error) {
	actor, store, err := actingOrganization(ctx)
	if err != nil {
	    return db.Team{}, err
	}
	return NewTeamService(store).RenameTeam(ctx, actor.AccountID, actor.SessionID, teamID, name)
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
		return addTeamMemberTx(ctx, q, actorAccountID, actingSessionID, teamID, personID)
	})
}

func (s *TeamService) RemoveTeamUser(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID, personID int64) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		return removeTeamMemberTx(ctx, q, actorAccountID, actingSessionID, teamID, personID)
	})
}
