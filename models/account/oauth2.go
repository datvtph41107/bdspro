package account

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// CreateOAuth2Account atomically creates a passwordless account and its OIDC
// identity. This must ONLY be called after verifying an ID token and a verified
// email from a trusted issuer. It never links an existing account by email.
func CreateOAuth2Account(ctx context.Context, tx Transactioner, identity auth_model.OIDCIdentity) (db.Account, error) {
	key, err := identity.Key()
	if err != nil {
		return db.Account{}, err
	}
	if !identity.EmailVerified || identity.Email == "" {
		return db.Account{}, auth_model.ErrInvalidEmail
	}
	var created db.Account
	err = tx.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		created, err = q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: identity.Email,
		})
		if err != nil {
			return err
		}
		created, err = q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: created.ID, Email: identity.Email,
		})
		if err != nil {
			return err
		}
		_, err = q.LinkOAuth2Identity(ctx, db.LinkOAuth2IdentityParams{
			Issuer: key.Issuer, Subject: key.Subject, AccountID: created.ID,
		})
		return err
	})
	if err != nil {
		return db.Account{}, err
	}
	return created, nil
}
