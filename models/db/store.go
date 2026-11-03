package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store exposes SQLC queries and the existing transaction operations.
type Store interface {
	Querier
	PhoneOTPRepository
	GetAccountByID(ctx context.Context, id int64) (Account, error)
	GetAccountByEmail(ctx context.Context, email string) (Account, error)
	ExecTx(ctx context.Context, fn func(*Queries) error) error
	GetOrCreateVerifyEmailTx(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error)
	StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error
	CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (Account, error)
	BeginEmailLinkTx(ctx context.Context, accountID int64, email string, tokenHash []byte) error
	ConfirmEmailLinkTx(ctx context.Context, accountID int64, tokenHash []byte) error
	LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error
	GetGoogleAccountID(ctx context.Context, googleSub string) (int64, error)
}

type SQLStore struct {
	pool *pgxpool.Pool
	*Queries
}

func NewStore(pool *pgxpool.Pool) Store {
	return &SQLStore{pool: pool, Queries: New(pool)}
}

// ExecTx owns transaction creation, commit and rollback.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err = fn(New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ Store = (*SQLStore)(nil)
