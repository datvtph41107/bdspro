package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// VerifyEmail atomically consumes a single-use proof and marks the owning
// account's email as verified. Failed consumption leaves account state unchanged.
func VerifyEmail(ctx context.Context, tx Transactioner, arg db.ConsumeVerifyEmailParams) (db.Account, error) {
	return db.WithTxResult(ctx, tx, func(q *db.Queries) (db.Account, error) {
		proof, err := q.ConsumeVerifyEmail(ctx, arg)
		if err != nil {
			return db.Account{}, err
		}
		return q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: proof.AccountID, Email: proof.Email,
		})
	})
}
