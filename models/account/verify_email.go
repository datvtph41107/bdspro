package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// VerifyEmail atomically consumes a single-use proof and marks the owning
// account's email as verified. Failed consumption leaves account state unchanged.
func VerifyEmail(ctx context.Context, tx Transactioner, arg db.ConsumeVerifyEmailParams) (db.Account, error) {
	var verifiedAccount db.Account
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
		proof, err := q.ConsumeVerifyEmail(ctx, arg)
		if err != nil {
			return err
		}
		verifiedAccount, err = q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: proof.AccountID, Email: proof.Email,
		})
		return err
	})
	if err != nil {
		return db.Account{}, err
	}
	return verifiedAccount, nil
}
