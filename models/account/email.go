package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
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
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
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
	if emailLoginTaken(err) {
		return util.WithCause(ErrEmailLinkTaken, err)
	}
	return err
}
