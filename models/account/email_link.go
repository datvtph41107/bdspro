package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// BeginEmailLink serializes a phone-first account's email-link request.
// The current SQLC query owns the resend cooldown and address availability.
func BeginEmailLink(ctx context.Context, tx Transactioner, accountID int64, email string, hash []byte) error {
	return tx.ExecTx(ctx, func(q *db.Queries) error {
		_, err := q.LockPhoneOnlyEmailAccount(ctx, accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEmailLinkUnavailable
		}
		if err != nil {
			return err
		}
		taken, err := q.EmailLinkAddressTaken(ctx, email)
		if err != nil {
			return err
		}
		if taken {
			return ErrEmailLinkTaken
		}
		n, err := q.UpsertEmailLinkChallenge(ctx, db.UpsertEmailLinkChallengeParams{
			AccountID: accountID, Email: email, TokenHash: hash,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrEmailLinkCooldown
		}
		return nil
	})
}

// ConfirmEmailLink consumes the single-use challenge and adds a verified email
// in one transaction, leaving the existing Account and credentials intact.
func ConfirmEmailLink(ctx context.Context, tx Transactioner, accountID int64, hash []byte) error {
	return tx.ExecTx(ctx, func(q *db.Queries) error {
		email, err := q.ConsumeEmailLinkChallenge(ctx, db.ConsumeEmailLinkChallengeParams{
			AccountID: accountID, TokenHash: hash,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEmailLinkInvalid
		}
		if err != nil {
			return err
		}
		n, err := q.AttachVerifiedEmail(ctx, db.AttachVerifiedEmailParams{
			ID: accountID, Email: email,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrEmailLinkUnavailable
		}
		return nil
	})
}
