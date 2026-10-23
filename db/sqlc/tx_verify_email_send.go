package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrEmailAlreadyVerified means the verification task became unnecessary.
var ErrEmailAlreadyVerified = errors.New("email already verified")

// GetOrCreateVerifyEmailTx reuses one unexpired verification per account.
// The account row lock serializes concurrent tasks for the same account.
// This is a business transaction; Asynq still manages task retries in Redis.
func (s *SQLStore) GetOrCreateVerifyEmailTx(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error) {
	var record VerifyEmail
	err := s.execTx(ctx, func(q *Queries) error {
		var verified bool
		if err := q.db.QueryRow(ctx,
			"SELECT is_email_verified FROM accounts WHERE id = $1 FOR UPDATE",
			arg.AccountID,
		).Scan(&verified); err != nil {
			return err
		}
		if verified {
			return ErrEmailAlreadyVerified
		}

		err := q.db.QueryRow(ctx, `
SELECT id, account_id, email, secret_code, is_used, created_at, expired_at
FROM verify_emails
WHERE account_id = $1 AND email = $2 AND is_used = false AND expired_at > now()
ORDER BY id DESC
LIMIT 1
`, arg.AccountID, arg.Email).Scan(
			&record.ID, &record.AccountID, &record.Email, &record.SecretCode,
			&record.IsUsed, &record.CreatedAt, &record.ExpiredAt,
		)
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
