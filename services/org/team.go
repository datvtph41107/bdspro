// Package org coordinates cross-model organization business operations.
package org

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
)

// NewTeam follows the Gitea package-function pattern: a domain model is
// populated after its WithTx2 call succeeds. When nested, model values remain
// provisional until the outer transaction commits.
func NewTeam(ctx context.Context, team *organization_model.Team) error {
	if team == nil || !organization_model.ValidTeamName(team.Name) {
		return organization_model.ErrTeamNameInvalid
	}
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	saved, err := db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Team, error) {
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
	*team = *saved
	return nil
}

// UpdateTeam operates on the same organization.Team domain type as NewTeam.
// Neither HTTP actions nor test callers pass a Store, SQLC queries, or session
// parameters. When nested, fields are provisional until the parent commits.
func UpdateTeam(ctx context.Context, team *organization_model.Team) error {
	if team == nil || !organization_model.ValidTeamName(team.Name) {
		return organization_model.ErrTeamNameInvalid
	}
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	saved, err := db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Team, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		return organization_model.UpdateTeam(txCtx, actor.AccountID, team)
	})
	if err != nil {
		return err
	}
	*team = *saved
	return nil
}

// AddTeamMember checks the acting session in the same SQLC transaction as the
// current owner-scoped membership mutation.
func AddTeamMember(ctx context.Context, teamID, personID int64) (db.TeamMember, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return db.TeamMember{}, auth_model.ErrSessionUnavailable
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (db.TeamMember, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return db.TeamMember{}, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return db.TeamMember{}, err
		}
		return organization_model.AddTeamUser(txCtx, q, actor.AccountID, teamID, personID)
	})
}

// RemoveTeamMember serializes the session-row lock and owner predicate with
// the SQLC deletion and its audit intent.
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
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return err
		}
		return organization_model.RemoveTeamUserTx(txCtx, q, actor.AccountID, teamID, personID)
	})
}
