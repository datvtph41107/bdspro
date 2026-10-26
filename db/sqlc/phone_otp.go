package db

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrOTPInvalid  = errors.New("invalid or expired phone code")
	ErrOTPCooldown = errors.New("phone code temporarily unavailable")
)

// PhoneOTPRepository is a narrow consumer contract implemented by SQLStore.
// It does NOT create a second PostgreSQL pool, query executor or repository.
type PhoneOTPRepository interface {
	AccountIDForPhone(ctx context.Context, phone string) (int64, error)
	StartChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte, devCode string) error
	ConsumeChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte) error
}

func (s *SQLStore) AccountIDForPhone(ctx context.Context, phone string) (int64, error) {
	return s.Queries.GetPhoneAccountIDQuery(ctx, phone)
}

func (s *SQLStore) StartChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte, devCode string) error {
	affected, err := s.Queries.UpsertPhoneChallengeQuery(ctx, UpsertPhoneChallengeQueryParams{
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
func (s *SQLStore) ConsumeChallenge(ctx context.Context, phone, purpose string, accountID int64, mac []byte) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.Queries.WithTx(tx)

	challenge, err := q.LockPhoneChallengeQuery(ctx, LockPhoneChallengeQueryParams{
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
		if err := q.RecordPhoneChallengeFailureQuery(ctx, RecordPhoneChallengeFailureQueryParams{
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
		if err := q.BindPhoneCredentialQuery(ctx, BindPhoneCredentialQueryParams{
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

	if err := q.DeletePhoneChallengeQuery(ctx, DeletePhoneChallengeQueryParams{
		PhoneE164: phone, Purpose: purpose,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ PhoneOTPRepository = (*SQLStore)(nil)
