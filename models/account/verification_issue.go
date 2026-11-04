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
	return db.WithTxResult(ctx, tx, func(q *db.Queries) (db.VerifyEmail, error) {
		verified, err := q.LockVerifyEmailAccountQuery(ctx, arg.AccountID)
		if err != nil {
			return db.VerifyEmail{}, err
		}
		if verified {
			return db.VerifyEmail{}, ErrEmailAlreadyVerified
		}
		record, err := q.FindReusableVerifyEmailQuery(ctx, db.FindReusableVerifyEmailQueryParams{
			AccountID: arg.AccountID, Email: arg.Email,
		})
		if err == nil {
			return record, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.VerifyEmail{}, err
		}
		return q.CreateVerifyEmail(ctx, arg)
	})
}
