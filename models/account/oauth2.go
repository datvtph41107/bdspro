package account

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/validation"
)

// CreateOAuth2Account atomically creates a passwordless account and its OIDC
// identity. This must ONLY be called after verifying an ID token and a verified
// email from a trusted issuer. It never links an existing account by email.
func CreateOAuth2Account(ctx context.Context, tx Transactioner, identity auth_model.OIDCClaims) (db.Account, error) {
	key, err := identity.Key()
	if err != nil {
		return db.Account{}, err
	}
	email, valid := validation.EmailAddress(identity.Email)
	if !identity.EmailVerified || !valid {
		return db.Account{}, auth_model.ErrInvalidEmail
	}
	created, err := db.WithTxResult(ctx, tx, func(q *db.Queries) (db.Account, error) {
		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return db.Account{}, err
		}
		account, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: email,
		})
		if err != nil {
			return db.Account{}, err
		}
		if err := q.CreateAccountSettings(ctx, account.ID); err != nil {
			return db.Account{}, err
		}
		account, err = q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: account.ID, Email: email,
		})
		if err != nil {
			return db.Account{}, err
		}
		_, err = q.LinkOAuth2Credential(ctx, db.LinkOAuth2CredentialParams{
			Issuer: key.Issuer, Subject: key.Subject, AccountID: account.ID,
		})
		return account, err
	})
	if emailLoginTaken(err) {
		return db.Account{}, util.WithCause(ErrEmailRegistered, err)
	}
	return created, err
}
