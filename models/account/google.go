package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// CreateGoogleAccount atomically creates a passwordless account and a verified
// Google identity. It never links an existing account by email.
func CreateGoogleAccount(ctx context.Context, tx Transactioner, email, googleSub string) (db.Account, error) {
	var created db.Account
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		created, err = q.CreateVerifiedGoogleAccount(ctx, db.CreateVerifiedGoogleAccountParams{
			PersonID: partyID, Email: email,
		})
		if err != nil {
			return err
		}
		_, err = q.LinkGoogleIdentityQuery(ctx, db.LinkGoogleIdentityQueryParams{
			AccountID: created.ID, GoogleSub: googleSub,
		})
		return err
	})
	return created, err
}
