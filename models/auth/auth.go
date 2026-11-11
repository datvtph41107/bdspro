package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	oauth_model "github.com/datvtph41107/bdspro/models/oauth"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// ResetPassword is a credential-recovery transaction: the reset token,
// password, sessions and third-party OAuth grants commit or roll back together.
// Internal stateless access JWTs still follow the existing expiry policy.
func ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		tx, err := db.GetEngine(txCtx)
		if err != nil {
			return err
		}
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		ctx := txCtx
		// Read the immutable owner of the one-use proof without consuming it.
		// Lock the account BEFORE deleting any proof: concurrent recovery
		// requests for different proofs otherwise can deadlock while trying
		// to delete each other's proof row.
		var accountID int64
		err = tx.QueryRow(ctx, `SELECT account_id FROM password_reset_tokens
			WHERE token_hash=$1 AND expires_at>now()`, tokenHash).Scan(&accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPasswordResetInvalid
		}
		if err != nil {
			return err
		}
		if err := oauth_model.LockAccountForCredentialRecovery(ctx, tx, accountID); err != nil {
			return err
		}
		consumedID, err := q.ConsumePasswordResetToken(ctx, tokenHash)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && consumedID != accountID) {
			return ErrPasswordResetInvalid
		}
		if err != nil {
			return err
		}
		if err := q.UpsertPasswordCredential(ctx, db.UpsertPasswordCredentialParams{
			AccountID: accountID, PasswordHash: passwordHash,
		}); err != nil {
			return err
		}
		// Invalidate queued and in-flight password-reset intents from the old generation.
		// A proof insertion holds FOR SHARE on this same account row.
		if err := q.BumpPasswordResetVersion(ctx, accountID); err != nil {
			return err
		}
		if err := q.DeletePasswordResetTokensForAccount(ctx, accountID); err != nil {
			return err
		}
		if err := q.DeleteSessionsForAccount(ctx, accountID); err != nil {
			return err
		}
		return oauth_model.RevokeOAuthForCredentialRecovery(ctx, tx, accountID)
	})
}

// AccountIDForPhone returns the account with the verified phone credential.
func AccountIDForPhone(ctx context.Context, phone string) (int64, error) {
	q, err := db.GetQueries(ctx)
	if err != nil { return 0, err }
	return q.GetPhoneAccountIDQuery(ctx, phone)
}

// StartChallenge enforces the rate/cooldown predicate in the SQLC upsert.
func StartChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte, devCode string) error {
	q, err := db.GetQueries(ctx)
	if err != nil { return err }
	affected, err := q.UpsertPhoneChallengeQuery(ctx, db.UpsertPhoneChallengeQueryParams{
		PhoneE164: phone, Purpose: purpose, AccountID: accountID,
		CodeMac: mac, DevCode: devCode,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOTPCooldown
	}
	return nil
}

// ConsumeChallenge preserves the security invariant: a wrong OTP commits
// its failed-attempt counter, while all credential writes remain atomic.
func ConsumeChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte) error {
	invalidCode := false
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}

		challenge, err := q.LockPhoneChallengeQuery(txCtx, db.LockPhoneChallengeQueryParams{
			PhoneE164: phone, Purpose: purpose,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOTPInvalid
		}
		if err != nil {
			return err
		}

		now := time.Now()
		if challenge.AccountID != accountID || !now.Before(challenge.ExpiresAt) ||
			now.Before(challenge.LockedUntil) || challenge.FailedAttempts >= 5 {
			return ErrOTPInvalid
		}

		if !hmac.Equal(challenge.CodeMac, mac) {
			if err := q.RecordPhoneChallengeFailureQuery(txCtx, db.RecordPhoneChallengeFailureQueryParams{
				PhoneE164: phone, Purpose: purpose,
			}); err != nil {
				return err
			}
			invalidCode = true
			return nil // Commit the failed-attempt counter before returning ErrOTPInvalid.
		}

		switch purpose {
		case "bind":
			if err := q.BindPhoneCredentialQuery(txCtx, db.BindPhoneCredentialQueryParams{
				AccountID: accountID, PhoneE164: phone,
			}); err != nil {
				if phoneCredentialConflict(err) {
					return util.WithCause(ErrPhoneUnavailable, err)
				}
				return err
			}
		case "login":
			current, err := q.LockPhoneCredentialQuery(txCtx, phone)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOTPInvalid
			}
			if err != nil {
				return err
			}
			if current != accountID {
				return ErrOTPInvalid
			}
		default:
			return ErrOTPInvalid
		}

		return q.DeletePhoneChallengeQuery(txCtx, db.DeletePhoneChallengeQueryParams{
			PhoneE164: phone, Purpose: purpose,
		})
	})
	if err != nil {
		return err
	}
	if invalidCode {
		return ErrOTPInvalid
	}
	return nil
}

// StartPhoneSignup owns the cooldown predicate; it does not create an account.
func StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error {
	q, err := db.GetQueries(ctx)
	if err != nil { return err }
	n, err := q.UpsertPhoneSignup(ctx, db.UpsertPhoneSignupParams{
		PhoneE164: phone, CodeMac: mac, DevCode: devCode,
	})
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrOTPCooldown
	}
	return nil
}

// CompletePhoneSignup commits failed attempts on wrong codes and atomically
// creates the Party, Person, Account and PhoneCredential on successful proof.
func CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (db.Account, error) {
	var account db.Account
	invalidCode := false
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}

		row, err := q.LockPhoneSignup(txCtx, phone)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOTPInvalid
		}
		if err != nil {
			return err
		}
		now := time.Now()
		if !now.Before(row.ExpiresAt) || now.Before(row.LockedUntil) || row.FailedAttempts >= 5 {
			return ErrOTPInvalid
		}
		if !hmac.Equal(row.CodeMac, mac) {
			if err := q.FailPhoneSignup(txCtx, phone); err != nil {
				return err
			}
			invalidCode = true
			return nil // Commit failed attempts rather than rolling them back.
		}

		partyID, err := db.CreatePersonWithParty(txCtx, q)
		if err != nil {
			return err
		}
		created, err := q.CreatePhoneOnlyAccount(txCtx, partyID)
		if err != nil {
			return err
		}
		if err := q.CreateAccountSettings(txCtx, created.ID); err != nil {
			return err
		}
		if err := q.BindPhoneCredentialQuery(txCtx, db.BindPhoneCredentialQueryParams{
			AccountID: created.ID, PhoneE164: phone,
		}); err != nil {
			if phoneRegisteredConflict(err) {
				return util.WithCause(ErrPhoneRegistered, err)
			}
			return err
		}
		if err := q.DeletePhoneSignup(txCtx, phone); err != nil {
			return err
		}
		account = db.Account{
			ID: created.ID, PersonID: created.PersonID, Email: created.Email,
			CreatedAt: created.CreatedAt, IsEmailVerified: created.IsEmailVerified,
		}
		return nil
	})
	if err != nil {
		return db.Account{}, err
	}
	if invalidCode {
		return db.Account{}, ErrOTPInvalid
	}
	return account, nil
}
