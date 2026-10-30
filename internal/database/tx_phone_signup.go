package db

import (
	"context"
	"crypto/hmac"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *SQLStore) StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error {
	n, err := s.Queries.UpsertPhoneSignup(ctx, UpsertPhoneSignupParams{
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

// CompletePhoneSignup verifies possession and inserts identity + credential
// atomically; invalid code attempts commit the failure count.
func (s *SQLStore) CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback(ctx)
	q := s.Queries.WithTx(tx)
	row, err := q.LockPhoneSignup(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrOTPInvalid
	}
	if err != nil {
		return Account{}, err
	}
	now := time.Now()
	if !now.Before(row.ExpiresAt) || now.Before(row.LockedUntil) || row.FailedAttempts >= 5 {
		return Account{}, ErrOTPInvalid
	}
	if !hmac.Equal(row.CodeMac, mac) {
		if err := q.FailPhoneSignup(ctx, phone); err != nil {
			return Account{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Account{}, err
		}
		return Account{}, ErrOTPInvalid
	}
	partyID, err := q.CreateParty(ctx)
	if err != nil {
		return Account{}, err
	}
	if err := q.CreatePerson(ctx, partyID); err != nil {
		return Account{}, err
	}
	if err := q.CreatePersonProfile(ctx, partyID); err != nil {
		return Account{}, err
	}
	a, err := q.CreatePhoneOnlyAccount(ctx, partyID)
	if err != nil {
		return Account{}, err
	}
	if err := q.CreateAccountSettings(ctx, a.ID); err != nil {
		return Account{}, err
	}
	if err := q.BindPhoneCredentialQuery(ctx, BindPhoneCredentialQueryParams{AccountID: a.ID, PhoneE164: phone}); err != nil {
		return Account{}, err
	}
	if err := q.DeletePhoneSignup(ctx, phone); err != nil {
		return Account{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Account{}, err
	}
	return Account{ID: a.ID, PersonID: a.PersonID, Email: a.Email, CreatedAt: a.CreatedAt, IsEmailVerified: a.IsEmailVerified}, nil
}
