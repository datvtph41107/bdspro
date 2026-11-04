package organization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/datvtph41107/bdspro/modules/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreatedTeamInvite contains the raw secret once; never persist or log Token.
type CreatedTeamInvite struct {
	Invite db.TeamInvite
	Token  string
}

// CreateTeamInvite stores a hash, not the raw token. Delivery is the service's responsibility.
func CreateTeamInvite(ctx context.Context, store db.Store, actorAccountID, teamID int64, email string, ttl time.Duration) (CreatedTeamInvite, error) {
	var result CreatedTeamInvite
	email, valid := validation.EmailAddress(email)
	if !valid || ttl <= 0 {
		return result, ErrTeamInviteInvalid
	}
	token, err := secret.RandomHex(32)
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256([]byte(token))
	var invite db.TeamInvite
	err = store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeExpiredTeamInvites(ctx, db.RevokeExpiredTeamInvitesParams{
			TeamID: teamID, InvitedEmail: email,
		}); err != nil {
			return err
		}
		var createErr error
		invite, createErr = q.CreateTeamInviteForOwner(ctx, db.CreateTeamInviteForOwnerParams{
			ID: uuid.New(), TeamID: teamID, ActorAccountID: actorAccountID,
			InvitedEmail: email, TokenHash: hash[:], ExpiresAt: time.Now().Add(ttl),
		})
		return createErr
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrTeamOwnerRequired
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "team_invites_open_uq" {
		return result, errors.Join(ErrTeamInviteExists, err)
	}
	if err != nil {
		return result, err
	}
	return CreatedTeamInvite{Invite: invite, Token: token}, nil
}

// AcceptTeamInvite commits proof consumption and membership creation together.
// A second use, expired proof, or wrong verified email cannot succeed.
func AcceptTeamInvite(ctx context.Context, store db.Store, accountID int64, token string) (db.TeamMember, error) {
	var member db.TeamMember
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return member, ErrTeamInviteInvalid
	}
	hash := sha256.Sum256([]byte(token))
	err = store.ExecTx(ctx, func(q *db.Queries) error {
		teamID, err := q.ConsumeTeamInvite(ctx, db.ConsumeTeamInviteParams{
			TokenHash: hash[:], AccountID: accountID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTeamInviteInvalid
		}
		if err != nil {
			return err
		}
		member, err = q.InsertInvitedTeamUser(ctx, db.InsertInvitedTeamUserParams{
			AccountID: accountID, TeamID: teamID,
		})
		return err
	})
	if err != nil {
		return db.TeamMember{}, err
	}
	return member, nil
}

func RevokeTeamInvite(ctx context.Context, store db.Store, actorAccountID int64, inviteID uuid.UUID) error {
	n, err := store.RevokeTeamInviteForOwner(ctx, db.RevokeTeamInviteForOwnerParams{
		InviteID: inviteID, ActorAccountID: actorAccountID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTeamInviteInvalid
	}
	return nil
}
