package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrPasswordResetInvalid = errors.New("invalid or expired password reset")

// ResetPasswordTx consumes a single-use token, sets the password credential
// and revokes outstanding refresh sessions in one atomic operation.
func (s *SQLStore) ResetPasswordTx(ctx context.Context, tokenHash []byte, passwordHash string) error {
	return s.ExecTx(ctx, func(q *Queries) error {
		accountID, err := q.ConsumePasswordResetToken(ctx, tokenHash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPasswordResetInvalid
		}
		if err != nil {
			return err
		}
		if err := q.UpsertPasswordCredential(ctx, UpsertPasswordCredentialParams{
			AccountID: accountID, PasswordHash: passwordHash,
		}); err != nil {
			return err
		}
		if err := q.DeletePasswordResetTokensForAccount(ctx, accountID); err != nil {
			return err
		}
		return q.DeleteSessionsForAccount(ctx, accountID)
	})
}
