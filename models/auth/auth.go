package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
	db "github.com/datvtph41107/bdspro/models/db"
)

// ResetPassword consumes the one-use proof, updates the password and removes
// reset proofs and refresh sessions in the same PostgreSQL transaction.
// Access JWTs continue to expire according to the existing stateless policy.
// Transactioner retains the model's existing public name; db owns the contract.
type Transactioner = db.Transactioner

func ResetPassword(ctx context.Context, tx Transactioner, tokenHash []byte, passwordHash string) error {
	return tx.ExecTx(ctx, func(q *db.Queries) error {
		accountID, err := q.ConsumePasswordResetToken(ctx, tokenHash)
		if errors.Is(err, pgx.ErrNoRows) {
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
		if err := q.DeletePasswordResetTokensForAccount(ctx, accountID); err != nil {
			return err
		}
		return q.DeleteSessionsForAccount(ctx, accountID)
	})
}

// AccountIDForPhone returns the account with the verified phone credential.
func AccountIDForPhone(ctx context.Context, store db.Store, phone string) (int64, error) {
	return store.GetPhoneAccountIDQuery(ctx, phone)
}

// StartChallenge enforces the rate/cooldown predicate in the SQLC upsert.
func StartChallenge(ctx context.Context, store db.Store, phone, purpose string, accountID int64, mac []byte, devCode string) error {
	affected, err := store.UpsertPhoneChallengeQuery(ctx, db.UpsertPhoneChallengeQueryParams{
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
func ConsumeChallenge(ctx context.Context, store db.Store, phone, purpose string, accountID int64, mac []byte) error {
	invalidCode := false
	err := store.ExecTx(ctx, func(q *db.Queries) error {
		challenge, err := q.LockPhoneChallengeQuery(ctx, db.LockPhoneChallengeQueryParams{
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
			if err := q.RecordPhoneChallengeFailureQuery(ctx, db.RecordPhoneChallengeFailureQueryParams{
				PhoneE164: phone, Purpose: purpose,
			}); err != nil {
				return err
			}
			invalidCode = true
			return nil // Commit the failed-attempt counter before returning ErrOTPInvalid.
		}

		switch purpose {
		case "bind":
			if err := q.BindPhoneCredentialQuery(ctx, db.BindPhoneCredentialQueryParams{
				AccountID: accountID, PhoneE164: phone,
			}); err != nil {
				if phoneCredentialConflict(err) {
					return util.WithCause(ErrPhoneUnavailable, err)
				}
				return err
			}
		case "login":
			current, err := q.LockPhoneCredentialQuery(ctx, phone)
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

		return q.DeletePhoneChallengeQuery(ctx, db.DeletePhoneChallengeQueryParams{
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
func StartPhoneSignup(ctx context.Context, store db.Store, phone string, mac []byte, devCode string) error {
	n, err := store.UpsertPhoneSignup(ctx, db.UpsertPhoneSignupParams{
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
func CompletePhoneSignup(ctx context.Context, store db.Store, phone string, mac []byte) (db.Account, error) {
	var account db.Account
	invalidCode := false
	err := store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.LockPhoneSignup(ctx, phone)
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
			if err := q.FailPhoneSignup(ctx, phone); err != nil {
				return err
			}
			invalidCode = true
			return nil // Commit failed attempts rather than rolling them back.
		}

		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return err
		}
		created, err := q.CreatePhoneOnlyAccount(ctx, partyID)
		if err != nil {
			return err
		}
		if err := q.CreateAccountSettings(ctx, created.ID); err != nil {
			return err
		}
		if err := q.BindPhoneCredentialQuery(ctx, db.BindPhoneCredentialQueryParams{
			AccountID: created.ID, PhoneE164: phone,
		}); err != nil {
			if phoneRegisteredConflict(err) {
				return util.WithCause(ErrPhoneRegistered, err)
			}
			return err
		}
		if err := q.DeletePhoneSignup(ctx, phone); err != nil {
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
