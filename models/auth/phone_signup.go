package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

func (s *phoneStore) StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error {
	n, err := s.Store.UpsertPhoneSignup(ctx, db.UpsertPhoneSignupParams{
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
func (s *phoneStore) CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (db.Account, error) {
	tx, err := s.Begin(ctx)
	if err != nil {
		return db.Account{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	row, err := q.LockPhoneSignup(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Account{}, ErrOTPInvalid
	}
	if err != nil {
		return db.Account{}, err
	}
	now := time.Now()
	if !now.Before(row.ExpiresAt) || now.Before(row.LockedUntil) || row.FailedAttempts >= 5 {
		return db.Account{}, ErrOTPInvalid
	}
	if !hmac.Equal(row.CodeMac, mac) {
		if err := q.FailPhoneSignup(ctx, phone); err != nil {
			return db.Account{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return db.Account{}, err
		}
		return db.Account{}, ErrOTPInvalid
	}
	partyID, err := q.CreateParty(ctx)
	if err != nil {
		return db.Account{}, err
	}
	if err := q.CreatePerson(ctx, partyID); err != nil {
		return db.Account{}, err
	}
	a, err := q.CreatePhoneOnlyAccount(ctx, partyID)
	if err != nil {
		return db.Account{}, err
	}
	if err := q.BindPhoneCredentialQuery(ctx, db.BindPhoneCredentialQueryParams{AccountID: a.ID, PhoneE164: phone}); err != nil {
		return db.Account{}, err
	}
	if err := q.DeletePhoneSignup(ctx, phone); err != nil {
		return db.Account{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.Account{}, err
	}
	return db.Account{ID: a.ID, PersonID: a.PersonID, Email: a.Email, CreatedAt: a.CreatedAt, IsEmailVerified: a.IsEmailVerified}, nil
}
