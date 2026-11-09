package org

import (
	"context"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// CreateTeamInvite resolves the verified actor and Store once from the live
// request; the service method owns the single transaction and outbox intent.
// The signing key remains an explicit dependency, never request state.
func CreateTeamInvite(ctx context.Context, teamID int64, email string, ttl time.Duration, signingKey string) (org_model.CreatedTeamInvite, error) {
	actor, store, err := actingOrganization(ctx)
	if err != nil {
		return org_model.CreatedTeamInvite{}, err
	}
	return NewTeamService(store).CreateTeamInvite(ctx, actor.AccountID, actor.SessionID, teamID, email, ttl, signingKey)
}

// AcceptTeamInvite rejects a malformed proof before obtaining a database
// capability, then lets the transaction perform session and token checks.
func AcceptTeamInvite(ctx context.Context, token string) (db.TeamMember, error) {
	if !org_model.ValidTeamInviteProof(token) {
		return db.TeamMember{}, org_model.ErrTeamInviteInvalid
	}
	actor, store, err := actingOrganization(ctx)
	if err != nil {
		return db.TeamMember{}, err
	}
	return NewTeamService(store).AcceptTeamInvite(ctx, actor.AccountID, actor.SessionID, token)
}

// RevokeTeamInvite delegates one authenticated, owner-scoped commit.
func RevokeTeamInvite(ctx context.Context, teamID int64, inviteID uuid.UUID) error {
	actor, store, err := actingOrganization(ctx)
	if err != nil {
		return err
	}
	return NewTeamService(store).RevokeTeamInvite(ctx, actor.AccountID, actor.SessionID, teamID, inviteID)
}

// CreateTeamInvite gives creation, owner authorization, durable outbox intent
// and acting-session authority ONE commit boundary. Never send mail in this
// transaction; the existing outbox worker owns delivery after commit.
func (s *TeamService) CreateTeamInvite(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID int64, email string, ttl time.Duration, signingKey string) (org_model.CreatedTeamInvite, error) {
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (org_model.CreatedTeamInvite, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return org_model.CreatedTeamInvite{}, err
		}
		return org_model.CreateTeamInviteTx(ctx, q, actorAccountID, teamID, email, ttl, signingKey)
	})
}

// AcceptTeamInvite is a single-use proof consumption and membership insert.
// Reject malformed input without opening a transaction.
func (s *TeamService) AcceptTeamInvite(ctx context.Context, accountID int64, actingSessionID uuid.UUID, token string) (db.TeamMember, error) {
	if !org_model.ValidTeamInviteProof(token) {
		return db.TeamMember{}, org_model.ErrTeamInviteInvalid
	}
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.TeamMember, error) {
		if err := auth_model.RequireActingSession(ctx, q, accountID, actingSessionID); err != nil {
			return db.TeamMember{}, err
		}
		return org_model.AcceptTeamInviteTx(ctx, q, accountID, token)
	})
}

// RevokeTeamInvite checks the actor session and the owner's current
// organization membership in the same transaction as the SQLC UPDATE.
func (s *TeamService) RevokeTeamInvite(ctx context.Context, actorAccountID int64, actingSessionID uuid.UUID, teamID int64, inviteID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return err
		}
		return org_model.RevokeTeamInvite(ctx, q, actorAccountID, teamID, inviteID)
	})
}
