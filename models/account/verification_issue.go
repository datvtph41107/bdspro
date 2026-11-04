package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// GetOrCreateVerifyEmail locks the account row to serialize verification
// workers. Retry after failed SMTP delivery reuses the still-valid proof.
func GetOrCreateVerifyEmail(ctx context.Context, tx Transactioner, arg db.CreateVerifyEmailParams) (db.VerifyEmail, error) {
	var record db.VerifyEmail
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
		verified, err := q.LockVerifyEmailAccountQuery(ctx, arg.AccountID)
		if err != nil {
			return err
		}
		if verified {
			return ErrEmailAlreadyVerified
		}
		record, err = q.FindReusableVerifyEmailQuery(ctx, db.FindReusableVerifyEmailQueryParams{
			AccountID: arg.AccountID, Email: arg.Email,
		})
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		record, err = q.CreateVerifyEmail(ctx, arg)
		return err
	})
	return record, err
}
