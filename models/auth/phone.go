package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

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
