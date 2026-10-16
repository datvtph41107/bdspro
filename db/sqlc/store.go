package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute database queries and transactions.
type Store struct {
	db *pgxpool.Pool
	*Queries
}

// NewStore creates a new Store.
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction.
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		// Use a live context for rollback even if the request was cancelled.
		if rbErr := tx.Rollback(context.Background()); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
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

// RegisterTx creates one Party, Person, Account and verification token atomically.
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
