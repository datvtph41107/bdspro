package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

func (s *phoneStore) AccountIDForPhone(ctx context.Context, phone string) (int64, error) {
	return s.Store.GetPhoneAccountIDQuery(ctx, phone)
}

func (s *phoneStore) StartChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte, devCode string) error {
	affected, err := s.Store.UpsertPhoneChallengeQuery(ctx, db.UpsertPhoneChallengeQueryParams{
		PhoneE164: phone, Purpose: purpose, AccountID: accountID, CodeMac: mac, DevCode: devCode,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrOTPCooldown
	}
	return nil
}

// ConsumeChallenge retains the locked-row and commit-on-wrong-code contract.
// The complete SQL source lives in db/query/phone_otp.sql; no SQL string here.
func (s *phoneStore) ConsumeChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte) error {
	tx, err := s.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)

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
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return ErrOTPInvalid // The failed-attempt counter is already committed.
	}

	switch purpose {
	case "bind":
		if err := q.BindPhoneCredentialQuery(ctx, db.BindPhoneCredentialQueryParams{
			AccountID: accountID, PhoneE164: phone,
		}); err != nil {
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

	if err := q.DeletePhoneChallengeQuery(ctx, db.DeletePhoneChallengeQueryParams{
		PhoneE164: phone, Purpose: purpose,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ PhoneOperations = (*phoneStore)(nil)
