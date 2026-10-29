package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store collects the generated queries and the domain transaction.
type Store interface {
	Querier
	PhoneOTPRepository
 GetAccountByID(ctx context.Context, id int64) (Account, error)
 GetAccountByEmail(ctx context.Context, email string) (Account, error)
	RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error)
	ExecTx(ctx context.Context, fn func(*Queries) error) error
	GetOrCreateVerifyEmailTx(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error)
	CreateOrganizationTx(ctx context.Context, arg CreateOrganizationTxParams) (Organization, error)
	VerifyEmailTx(ctx context.Context, arg ConsumeVerifyEmailParams) (Account, error)
	StartPhoneSignup(ctx context.Context, phone string, mac []byte, devCode string) error
	CompletePhoneSignup(ctx context.Context, phone string, mac []byte) (Account, error)
	ResetPasswordTx(ctx context.Context, tokenHash []byte, passwordHash string) error
	LinkGoogleIdentity(ctx context.Context, arg LinkGoogleIdentityParams) error
	GetGoogleAccountID(ctx context.Context, googleSub string) (int64, error)
	CreateGoogleAccountTx(ctx context.Context, email, googleSub string) (Account, error)
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

type RegisterTxParams struct {
	Email        string
	PasswordHash string
}

// RegisterTx creates Party, Person and Account atomically.
func (s *SQLStore) RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error) {
	var account Account
	err := s.ExecTx(ctx, func(q *Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err = q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		account, err = q.CreateAccount(ctx, CreateAccountParams{
			PersonID: partyID, Email: arg.Email,
		})
		if err != nil {
			return err
		}
		if err := q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
			AccountID: account.ID, PasswordHash: arg.PasswordHash,
		}); err != nil {
			return err
		}
		return q.RecordEmailVerificationJob(ctx, account.ID)
	})
	return account, err
}

var _ Store = (*SQLStore)(nil)
