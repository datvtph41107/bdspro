package organization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreatedTeamInvite contains the raw secret once; never persist or log Token.
type CreatedTeamInvite struct {
	Invite db.TeamInvite
	Token  string
}

// CreateTeamInvite atomically stores the hashed proof and a durable delivery intent.
// The same secret can be recovered by the worker from the random invitation ID
// and the signing key, without storing a plaintext proof in the outbox.
func CreateTeamInvite(ctx context.Context, store db.Store, actorAccountID, teamID int64, email string, ttl time.Duration, signingKey string) (CreatedTeamInvite, error) {
	var result CreatedTeamInvite
	email, valid := validation.EmailAddress(email)
	if !valid || ttl <= 0 {
		return result, ErrTeamInviteInvalid
	}
	inviteID := uuid.New()
	token, err := secret.DeriveToken(signingKey, "team-invite:v1", inviteID[:])
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256([]byte(token))
	invite, err := db.WithTxResult(ctx, store, func(q *db.Queries) (db.TeamInvite, error) {
		if err := q.RevokeExpiredTeamInvites(ctx, db.RevokeExpiredTeamInvitesParams{
			TeamID: teamID, InvitedEmail: email,
		}); err != nil {
			return db.TeamInvite{}, err
		}
		record, err := q.CreateTeamInviteForOwner(ctx, db.CreateTeamInviteForOwnerParams{
			ID: inviteID, TeamID: teamID, ActorAccountID: actorAccountID,
			InvitedEmail: email, TokenHash: hash[:], ExpiresAt: time.Now().Add(ttl),
		})
		if err != nil {
			return db.TeamInvite{}, err
		}
		return record, q.RecordTeamInviteJob(ctx, record.ID)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return result, util.WithCause(ErrTeamOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "team_invites_open_uq") {
		return result, util.WithCause(ErrTeamInviteExists, err)
	}
	if err != nil {
		return result, err
	}
	return CreatedTeamInvite{Invite: invite, Token: token}, nil
}

// AcceptTeamInvite commits proof consumption and membership creation together.
// A second use, expired proof, or wrong verified email cannot succeed.
func AcceptTeamInvite(ctx context.Context, store db.Store, accountID int64, token string) (db.TeamMember, error) {
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return db.TeamMember{}, ErrTeamInviteInvalid
	}
	hash := sha256.Sum256([]byte(token))
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.TeamMember, error) {
		teamID, err := q.ConsumeTeamInvite(ctx, db.ConsumeTeamInviteParams{
			TokenHash: hash[:], AccountID: accountID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return db.TeamMember{}, ErrTeamInviteInvalid
		}
		if err != nil {
			return db.TeamMember{}, err
		}
		return q.InsertInvitedTeamUser(ctx, db.InsertInvitedTeamUserParams{
			AccountID: accountID, TeamID: teamID,
		})
	})
}

func RevokeTeamInvite(ctx context.Context, store db.Store, actorAccountID, teamID int64, inviteID uuid.UUID) error {
	n, err := store.RevokeTeamInviteForOwner(ctx, db.RevokeTeamInviteForOwnerParams{
		InviteID: inviteID, ActorAccountID: actorAccountID, TeamID: teamID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTeamInviteInvalid
	}
	return nil
}
