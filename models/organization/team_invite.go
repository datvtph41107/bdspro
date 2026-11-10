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

// TeamInvite is an invitation model without the SQLC proof hash or
// internal state columns. Consumers need only these public metadata fields.
type TeamInvite struct {
	ID                 uuid.UUID
	TeamID             int64
	InvitedEmail       string
	InvitedByAccountID int64
	ExpiresAt          time.Time
	CreatedAt          time.Time
}

func TeamInviteFromDB(row db.TeamInvite) TeamInvite {
	return TeamInvite{
		ID: row.ID, TeamID: row.TeamID,
		InvitedEmail: row.InvitedEmail, InvitedByAccountID: row.InvitedByAccountID,
		ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt,
	}
}

// CreatedTeamInvite carries a proof transiently for delivery, never JSON.
type CreatedTeamInvite struct {
	Invite TeamInvite
	Token  string `json:"-"`
}

// CreateTeamInvite atomically stores the hashed proof and a durable delivery intent.
// The same secret can be recovered by the worker from the random invitation ID
// and the signing key, without storing a plaintext proof in the outbox.
func CreateTeamInvite(ctx context.Context, store db.Store, actorAccountID, teamID int64, email string, ttl time.Duration, signingKey string) (CreatedTeamInvite, error) {
	// Reject obviously invalid invitations before acquiring a connection.
	if _, valid := validation.EmailAddress(email); !valid || ttl <= 0 {
		return CreatedTeamInvite{}, ErrTeamInviteInvalid
	}
	// Validate the signing-key contract before acquiring a DB transaction.
	// The transactional variant still validates through DeriveToken.
	if !secret.ValidDerivationKey(signingKey) {
		return CreatedTeamInvite{}, ErrTeamInviteInvalid
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (CreatedTeamInvite, error) {
		return CreateTeamInviteTx(ctx, q, actorAccountID, teamID, email, ttl, signingKey)
	})
}

// CreateTeamInviteTx stores proof metadata and its durable outbox intent
// together. The caller owns the transaction; do not call ExecTx here.
func CreateTeamInviteTx(ctx context.Context, q *db.Queries, actorAccountID, teamID int64, email string, ttl time.Duration, signingKey string) (CreatedTeamInvite, error) {
	email, valid := validation.EmailAddress(email)
	if !valid || ttl <= 0 {
		return CreatedTeamInvite{}, ErrTeamInviteInvalid
	}
	inviteID := uuid.New()
	token, err := secret.DeriveToken(signingKey, "team-invite:v1", inviteID[:])
	if err != nil {
		return CreatedTeamInvite{}, err
	}
	hash := sha256.Sum256([]byte(token))
	if err := q.RevokeExpiredTeamInvites(ctx, db.RevokeExpiredTeamInvitesParams{
		TeamID: teamID, InvitedEmail: email,
	}); err != nil {
		return CreatedTeamInvite{}, err
	}
	record, err := q.CreateTeamInviteForOwner(ctx, db.CreateTeamInviteForOwnerParams{
		ID: inviteID, TeamID: teamID, ActorAccountID: actorAccountID,
		InvitedEmail: email, TokenHash: hash[:], ExpiresAt: time.Now().Add(ttl),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return CreatedTeamInvite{}, util.WithCause(ErrTeamOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "team_invites_open_uq") {
		return CreatedTeamInvite{}, util.WithCause(ErrTeamInviteExists, err)
	}
	if err != nil {
		return CreatedTeamInvite{}, err
	}
	if err := q.RecordTeamInviteJob(ctx, record.ID); err != nil {
		return CreatedTeamInvite{}, err
	}
	return CreatedTeamInvite{Invite: TeamInviteFromDB(record), Token: token}, nil
}

// AcceptTeamInvite commits proof consumption and membership creation together.
// A second use, expired proof, or wrong verified email cannot succeed.
// ValidTeamInviteProof checks only proof encoding. It grants no authority:
// SQLC verifies recipient identity, active membership, expiry and single use.
func ValidTeamInviteProof(token string) bool {
	raw, err := hex.DecodeString(token)
	return err == nil && len(raw) == 32
}

func AcceptTeamInvite(ctx context.Context, store db.Store, accountID int64, token string) (*TeamMember, error) {
	if !ValidTeamInviteProof(token) {
		return nil, ErrTeamInviteInvalid
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (*TeamMember, error) {
		return AcceptTeamInviteTx(ctx, q, accountID, token)
	})
}

// AcceptTeamInviteTx consumes the proof and creates membership in the same
// transaction owned by its caller. On any failure, proof consumption rolls back.
func AcceptTeamInviteTx(ctx context.Context, q *db.Queries, accountID int64, token string) (*TeamMember, error) {
	if !ValidTeamInviteProof(token) {
		return nil, ErrTeamInviteInvalid
	}
	hash := sha256.Sum256([]byte(token))
	teamID, err := q.ConsumeTeamInvite(ctx, db.ConsumeTeamInviteParams{
		TokenHash: hash[:], AccountID: accountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTeamInviteInvalid
	}
	if err != nil {
		return nil, err
	}
	member, err := q.InsertInvitedTeamUser(ctx, db.InsertInvitedTeamUserParams{
		AccountID: accountID, TeamID: teamID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// A revoked organization membership invalidates the invitation.
		return nil, ErrTeamInviteInvalid
	}
	if err != nil {
		return nil, err
	}
	return TeamMemberFromDB(member), nil
}

// TeamInviteRevoker is consumed by the model. Both a top-level store and
// transaction-bound SQLC Queries implement this single operation.
type TeamInviteRevoker interface {
	RevokeTeamInviteForOwner(context.Context, db.RevokeTeamInviteForOwnerParams) (int64, error)
}

func RevokeTeamInvite(ctx context.Context, writer TeamInviteRevoker, actorAccountID, teamID int64, inviteID uuid.UUID) error {
	n, err := writer.RevokeTeamInviteForOwner(ctx, db.RevokeTeamInviteForOwnerParams{
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
