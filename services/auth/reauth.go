package auth

import (
	"context"
	"errors"
	"strings"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/jackc/pgx/v5"
)

var ErrReauthRequired = util.WithPublic(util.ErrPermissionDenied, "recent authentication missing or invalid", "reauthentication_required", "recent authentication required")
var ErrReauthAction = util.WithPublic(util.ErrInvalidArgument, "invalid reauthentication action", "validation_failed", "invalid reauthentication action")

type RecentAuthResult struct {
	Proof     string `json:"reauthentication_proof"`
	Action    string `json:"action"`
	ExpiresIn int    `json:"expires_in"`
}

func passwordReauthAction(action string) bool {
	switch action {
	case "email.link", "phone.link", "google.link", "email.unlink", "phone.unlink", "google.unlink", "password.unlink", "sessions.revoke_all", "organization.owner.transfer":
		return true
	default:
		return false
	}
}

func ReauthenticatePassword(ctx context.Context, actor authmodel.Actor, password, action string) (RecentAuthResult, error) {
	if !passwordReauthAction(action) {
		return RecentAuthResult{}, ErrReauthAction
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return RecentAuthResult{}, err
	}
	old, err := q.GetPasswordHash(ctx, actor.AccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RecentAuthResult{}, rejectMissingPassword(password)
	}
	if err != nil {
		return RecentAuthResult{}, err
	}
	if err := token.CheckPassword(password, old); err != nil {
		if errors.Is(err, token.ErrPasswordBusy) {
			return RecentAuthResult{}, err
		}
		return RecentAuthResult{}, ErrInvalidCredentials
	}
	if _, err := q.DeleteExpiredRecentAuthProofs(ctx, 100); err != nil {
		return RecentAuthResult{}, err
	}
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return RecentAuthResult{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (RecentAuthResult, error) {
		q, err := LockActorForMutation(txCtx, actor)
		if err != nil {
			return RecentAuthResult{}, err
		}
		current, err := q.GetPasswordHash(txCtx, actor.AccountID)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && current != old {
			return RecentAuthResult{}, ErrInvalidCredentials
		}
		if err != nil {
			return RecentAuthResult{}, err
		}
		if err := q.CreateRecentAuthProof(txCtx, db.CreateRecentAuthProofParams{
			Digest: digest, AccountID: actor.AccountID, SessionID: actor.SessionID,
			Action: action, Method: "password",
		}); err != nil {
			return RecentAuthResult{}, err
		}
		return RecentAuthResult{Proof: raw, Action: action, ExpiresIn: 600}, nil
	})
}

func RequestPasswordSetProof(ctx context.Context, actor authmodel.Actor) error {
	if setting.Common.RunMode != "development" {
		return ErrReauthRequired
	}
	if !setting.Mailer.Enabled() {
		return mailer.ErrUnavailable
	}
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return err
	}
	email, err := db.WithTx2(ctx, func(txCtx context.Context) (string, error) {
		q, err := LockActorForMutation(txCtx, actor)
		if err != nil {
			return "", err
		}
		u, err := GetCurrentUser(txCtx, actor.AccountID)
		if err != nil {
			return "", err
		}
		if u.Email == nil || !u.Credentials.EmailVerified || !u.Credentials.GoogleLinked || u.Credentials.PasswordSet {
			return "", ErrReauthRequired
		}
		if err := q.DeleteAccountRecentAuthProofs(txCtx, actor.AccountID); err != nil {
			return "", err
		}
		if err := q.CreateRecentAuthProof(txCtx, db.CreateRecentAuthProofParams{
			Digest: digest, AccountID: actor.AccountID, SessionID: actor.SessionID,
			Action: "password.set", Method: "verified_email", EmailSnapshot: *u.Email,
		}); err != nil {
			return "", err
		}
		return *u.Email, nil
	})
	if err != nil {
		return err
	}
	return mailer.SendPasswordSetProof(ctx, email, raw)
}

func ConsumeRecentAuthProof(ctx context.Context, q *db.Queries, actor authmodel.Actor, raw, action string) error {
	if !db.InTransaction(ctx) {
		return ErrReauthRequired
	}
	digest, err := token.RecoveryDigest(raw)
	if err != nil {
		return ErrReauthRequired
	}
	row, err := q.ConsumeRecentAuthProof(ctx, db.ConsumeRecentAuthProofParams{
		Digest: digest, AccountID: actor.AccountID, SessionID: actor.SessionID, Action: action,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReauthRequired
	}
	if err != nil {
		return err
	}
	if row.Method == "password" && passwordReauthAction(action) {
		return nil
	}
	if row.Method == "google" && googleReauthAction(action) {
		owner, err := googleOwner(ctx, q, row.GoogleSubject, true)
		if err != nil {
			return err
		}
		if owner == actor.AccountID {
			return nil
		}
		return ErrReauthRequired
	}
	if row.Method == "verified_email" && action == "password.set" && setting.Common.RunMode == "development" {
		u, err := GetCurrentUser(ctx, actor.AccountID)
		if err != nil {
			return err
		}
		if u.Email != nil && strings.EqualFold(*u.Email, row.EmailSnapshot) && u.Credentials.EmailVerified &&
			u.Credentials.GoogleLinked && !u.Credentials.PasswordSet {
			return nil
		}
	}
	return ErrReauthRequired
}
