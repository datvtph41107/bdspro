package account

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
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
	var created db.Account
	err = tx.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return err
		}
		created, err = q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: email,
		})
		if err != nil {
			return err
		}
		created, err = q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: created.ID, Email: email,
		})
		if err != nil {
			return err
		}
		_, err = q.LinkOAuth2Credential(ctx, db.LinkOAuth2CredentialParams{
			Issuer: key.Issuer, Subject: key.Subject, AccountID: created.ID,
		})
		return err
	})
	if err != nil {
		return db.Account{}, err
	}
	return created, nil
}
