// Package auth owns persisted authentication credentials and proofs.
package auth

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// ResetPassword consumes the one-use proof, updates the password and removes
// reset proofs and refresh sessions in the same PostgreSQL transaction.
// Access JWTs continue to expire according to the existing stateless policy.
type Transactioner interface {
	ExecTx(context.Context, func(*db.Queries) error) error
}

func ResetPassword(ctx context.Context, tx Transactioner, tokenHash []byte, passwordHash string) error {
	return tx.ExecTx(ctx, func(q *db.Queries) error {
		accountID, err := q.ConsumePasswordResetToken(ctx, tokenHash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPasswordResetInvalid
		}
		if err != nil {
			return err
		}
		if err := q.UpsertPasswordCredential(ctx, db.UpsertPasswordCredentialParams{
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
