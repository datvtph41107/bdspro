package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

var (
	ErrEmailLinkUnavailable = errors.New("account cannot attach an email")
	ErrEmailLinkTaken       = errors.New("email address belongs to another account")
	ErrEmailLinkCooldown    = errors.New("email verification temporarily unavailable")
	ErrEmailLinkInvalid     = errors.New("invalid or expired email link")
)

func (s *SQLStore) BeginEmailLinkTx(ctx context.Context, accountID int64, email string, hash []byte) error {
	return s.ExecTx(ctx, func(q *Queries) error {
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
		n, err := q.UpsertEmailLinkChallenge(ctx, UpsertEmailLinkChallengeParams{
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

func (s *SQLStore) ConfirmEmailLinkTx(ctx context.Context, accountID int64, hash []byte) error {
	return s.ExecTx(ctx, func(q *Queries) error {
		email, err := q.ConsumeEmailLinkChallenge(ctx, ConsumeEmailLinkChallengeParams{
			AccountID: accountID, TokenHash: hash,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEmailLinkInvalid
		}
		if err != nil {
			return err
		}
		n, err := q.AttachVerifiedEmail(ctx, AttachVerifiedEmailParams{ID: accountID, Email: email})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrEmailLinkUnavailable
		}
		return nil
	})
}
