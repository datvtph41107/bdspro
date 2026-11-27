package auth

import (
	"context"
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/services/mailer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRecoveryInvalid = util.WithPublic(util.ErrInvalidArgument, "recovery proof invalid", "invalid_proof", "invalid or expired recovery proof")
	ErrPasswordInvalid = util.WithPublic(util.ErrInvalidArgument, "invalid password policy", "validation_failed", "password does not meet policy")
	ErrEmailInvalid    = util.WithPublic(util.ErrInvalidArgument, "invalid email", "validation_failed", "invalid email address")
)

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) > 254 || strings.ContainsAny(email, "\r\n\t") {
		return "", ErrEmailInvalid
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrEmailInvalid
	}
	return email, nil
}

func newPasswordHash(password string) (string, error) {
	hash, err := token.HashPassword(password)
	if errors.Is(err, token.ErrPasswordPolicy) {
		return "", ErrPasswordInvalid
	}
	return hash, err
}

func RequestPasswordReset(ctx context.Context, email string) error {
	start := time.Now()
	work, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	err := requestPasswordReset(work, email)
	if err != nil {
		log.Ctx(ctx).Warn().Msg("auth.password_reset.request_not_delivered")
	}
	timer := time.NewTimer(max(0, 4*time.Second-time.Since(start)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func requestPasswordReset(ctx context.Context, email string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	account, err := q.ReadAccountByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, digest, err := token.NewRecoveryProof()
	if err != nil {
		return err
	}
	issued, err := db.WithTx2(ctx, func(txCtx context.Context) (bool, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return false, err
		}
		if _, err = q.LockSessionRevocationAccount(txCtx, account.ID); errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		current, err := q.ReadAccountByID(txCtx, account.ID)
		if err != nil {
			return false, err
		}
		if err := requireActiveAccount(txCtx, q, account.ID); errors.Is(err, util.ErrUnauthenticated) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		if !current.IsEmailVerified || strings.ToLower(strings.TrimSpace(current.Email)) != email {
			return false, nil
		}
		if _, err := q.GetPasswordHash(txCtx, account.ID); errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		if err := q.CreatePasswordResetToken(txCtx, db.CreatePasswordResetTokenParams{AccountID: account.ID, TokenHash: digest}); err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil || !issued {
		return err
	}
	return mailer.SendPasswordReset(ctx, email, raw)
}

func ResetPassword(ctx context.Context, raw, password string) error {
	digest, err := token.RecoveryDigest(raw)
	if err != nil {
		return ErrRecoveryInvalid
	}
	hash, err := newPasswordHash(password)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		engine, err := db.GetEngine(txCtx)
		if err != nil {
			return err
		}
		var accountID int64

		err = engine.QueryRow(txCtx, "SELECT account_id FROM password_reset_tokens WHERE token_hash=$1 AND expires_at>clock_timestamp()", digest).Scan(&accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRecoveryInvalid
		}
		if err != nil {
			return err
		}
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		if _, err = q.LockSessionRevocationAccount(txCtx, accountID); errors.Is(err, pgx.ErrNoRows) {
			return ErrRecoveryInvalid
		} else if err != nil {
			return err
		}
		account, err := q.ReadAccountByID(txCtx, accountID)
		if err != nil {
			return err
		}
		if err := requireActiveAccount(txCtx, q, accountID); errors.Is(err, util.ErrUnauthenticated) {
			return ErrRecoveryInvalid
		} else if err != nil {
			return err
		}
		if !account.IsEmailVerified || account.Email == "" {
			return ErrRecoveryInvalid
		}
		if _, err = q.GetPasswordHash(txCtx, accountID); errors.Is(err, pgx.ErrNoRows) {
			return ErrRecoveryInvalid
		} else if err != nil {
			return err
		}

		var owner int64
		err = engine.QueryRow(txCtx, "SELECT account_id FROM password_reset_tokens WHERE token_hash=$1 AND expires_at>clock_timestamp()", digest).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && owner != accountID {
			return ErrRecoveryInvalid
		}
		if err != nil {
			return err
		}
		consumed, err := q.ConsumePasswordResetToken(txCtx, digest)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && consumed != accountID {
			return ErrRecoveryInvalid
		}
		if err != nil {
			return err
		}
		return replacePassword(txCtx, q, accountID, hash, "auth.password.reset")
	})
}

func ChangePassword(ctx context.Context, actor authmodel.Actor, current, password, clientIP, agent string) (LoginResult, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	old, err := q.GetPasswordHash(ctx, actor.AccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, rejectMissingPassword(current)
	}
	if err != nil {
		return LoginResult{}, err
	}
	if err := token.CheckPassword(current, old); err != nil {
		if errors.Is(err, token.ErrPasswordBusy) {
			return LoginResult{}, err
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	hash, err := newPasswordHash(password)
	if err != nil {
		return LoginResult{}, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := lockActor(txCtx, actor)
		if err != nil {
			return LoginResult{}, err
		}
		saved, err := q.GetPasswordHash(txCtx, actor.AccountID)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && saved != old {
			return LoginResult{}, ErrInvalidCredentials
		}
		if err != nil {
			return LoginResult{}, err
		}
		if err := replacePassword(txCtx, q, actor.AccountID, hash, "auth.password.changed"); err != nil {
			return LoginResult{}, err
		}

		return issueSession(txCtx, q, actor.AccountID, clientIP, agent, false)
	})
}

func LockActorForMutation(ctx context.Context, actor authmodel.Actor) (*db.Queries, error) {
	return lockActor(ctx, actor)
}

func lockActor(ctx context.Context, actor authmodel.Actor) (*db.Queries, error) {
	if actor.AccountID <= 0 || actor.SessionID == uuid.Nil || !db.InTransaction(ctx) {
		return nil, util.ErrUnauthenticated
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = q.LockSessionRevocationAccount(ctx, actor.AccountID); errors.Is(err, pgx.ErrNoRows) {
		return nil, util.ErrUnauthenticated
	} else if err != nil {
		return nil, err
	}
	if err := requireActiveAccount(ctx, q, actor.AccountID); err != nil {
		return nil, err
	}
	if _, err = q.LockActingSession(ctx, db.LockActingSessionParams{AccountID: actor.AccountID, SessionID: actor.SessionID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, util.ErrUnauthenticated
	} else if err != nil {
		return nil, err
	}
	return q, nil
}

func replacePassword(ctx context.Context, q *db.Queries, accountID int64, hash, action string) error {
	if err := q.UpsertPasswordCredential(ctx, db.UpsertPasswordCredentialParams{AccountID: accountID, PasswordHash: hash}); err != nil {
		return err
	}
	if err := invalidateCredentialAuthority(ctx, q, accountID); err != nil {
		return err
	}
	id := strconv.FormatInt(accountID, 10)
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action: action, ActorAccountID: accountID,
		ScopeType: "account", ScopeRef: id, TargetType: "password", TargetRef: id,
	})
}
