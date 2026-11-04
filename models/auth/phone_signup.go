package auth

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

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

		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		created, err := q.CreatePhoneOnlyAccount(ctx, partyID)
		if err != nil {
			return err
		}
		if err := q.BindPhoneCredentialQuery(ctx, db.BindPhoneCredentialQueryParams{
			AccountID: created.ID, PhoneE164: phone,
		}); err != nil {
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
