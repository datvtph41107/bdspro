package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store collects the generated queries and the domain transaction.
type Store interface {
	Querier
	PhoneOTPRepository
	RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error)
	GetOrCreateVerifyEmailTx(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error)
	CreateOrganizationTx(ctx context.Context, arg CreateOrganizationTxParams) (Organization, error)
	VerifyEmailTx(ctx context.Context, arg ConsumeVerifyEmailParams) (Account, error)
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

// execTx owns transaction creation, commit and rollback, like SimpleBank.
func (s *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
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

type RegisterTxParams struct {
	Email        string
	PasswordHash string
	AfterCreate  func(account Account) error
}

// RegisterTx creates Party, Person and Account atomically.
func (s *SQLStore) RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error) {
	var account Account
	err := s.execTx(ctx, func(q *Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err = q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		account, err = q.CreateAccount(ctx, CreateAccountParams{
			PersonID: partyID, Email: arg.Email, PasswordHash: arg.PasswordHash,
		})
		if err != nil {
			return err
		}
		if arg.AfterCreate != nil {
			return arg.AfterCreate(account)
		}
		return nil
	})
	return account, err
}

var _ Store = (*SQLStore)(nil)
