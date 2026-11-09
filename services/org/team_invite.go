package org

import (
	"context"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// CreateTeamInvite uses one request-scoped transaction for authorization,
// invite persistence and durable outbox. Mail delivery stays outside the commit.
func CreateTeamInvite(ctx context.Context, teamID int64, email string, ttl time.Duration, signingKey string) (org_model.CreatedTeamInvite, error) {
	actor, err := actingOrganization(ctx)
	if err != nil {
		return org_model.CreatedTeamInvite{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (org_model.CreatedTeamInvite, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return org_model.CreatedTeamInvite{}, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return org_model.CreatedTeamInvite{}, err
		}
		return org_model.CreateTeamInviteTx(txCtx, q, actor.AccountID, teamID, email, ttl, signingKey)
	})
}

// AcceptTeamInvite validates proof before opening any DB connection and then
// consumes the proof once under the verified actor's session lock.
func AcceptTeamInvite(ctx context.Context, token string) (db.TeamMember, error) {
	if !org_model.ValidTeamInviteProof(token) {
		return db.TeamMember{}, org_model.ErrTeamInviteInvalid
	}
	actor, err := actingOrganization(ctx)
	if err != nil {
		return db.TeamMember{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (db.TeamMember, error) {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return db.TeamMember{}, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return db.TeamMember{}, err
		}
		return org_model.AcceptTeamInviteTx(txCtx, q, actor.AccountID, token)
	})
}

// RevokeTeamInvite executes the session lock, owner authorization and SQLC
// mutation atomically.
func RevokeTeamInvite(ctx context.Context, teamID int64, inviteID uuid.UUID) error {
	actor, err := actingOrganization(ctx)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return err
		}
		return org_model.RevokeTeamInvite(txCtx, q, actor.AccountID, teamID, inviteID)
	})
}
