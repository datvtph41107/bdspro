package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store combines generated SQLC queries and transactions, as in SimpleBank.
// It owns database writes only; HTTP validation, password hashing and mail stay
// in the application service.
type Store struct {
	pool *pgxpool.Pool
	*Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, Queries: New(pool)}
}

// execTx makes the callback and its final deferred constraints one atomic unit.
// pgx commits on nil and rolls back on an error (including COMMIT errors).
func (s *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(s.WithTx(tx))
	})
}

type RegisterTxParams struct {
	Email        string
	PasswordHash string
	TokenHash    string
}

type RegisterTxResult struct {
	PartyID        int64
	AccountID      int64
	AccountEmailID int64
}

// RegisterTx is the BDSPro analogue of SimpleBank's TransferTx:
// multiple SQLC writes succeed together, or none remain.
func (s *Store) RegisterTx(ctx context.Context, arg RegisterTxParams) (RegisterTxResult, error) {
	var result RegisterTxResult
	err := s.execTx(ctx, func(q *Queries) error {
		var err error

		result.PartyID, err = q.CreatePersonParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePersonSubtype(ctx, result.PartyID); err != nil {
			return err
		}

		result.AccountID, err = q.CreateAccount(ctx, result.PartyID)
		if err != nil {
			return err
		}
		result.AccountEmailID, err = q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
			AccountID: result.AccountID, Email: arg.Email,
		})
		if err != nil {
			return err
		}
		if err := q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
			AccountID: result.AccountID, PasswordHash: arg.PasswordHash,
		}); err != nil {
			return err
		}
		return q.CreateEmailVerificationToken(ctx, CreateEmailVerificationTokenParams{
			AccountEmailID: result.AccountEmailID, TokenHash: arg.TokenHash,
		})
	})
	if err != nil {
		return RegisterTxResult{}, err
	}
	return result, nil
}
