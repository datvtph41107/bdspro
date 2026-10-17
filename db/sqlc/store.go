package db

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// ExecTx executes SQLC queries within a single database transaction.
// The callback must use the supplied Queries so every statement uses the same tx.
func (store *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()

	q := New(tx)
	err = fn(q)
	if err != nil {
		// Use a live context for rollback even if the request was cancelled.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		rbErr := tx.Rollback(cleanup)
		cancel()
		if rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback failed: %w", rbErr))
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
	err := s.ExecTx(ctx, func(q *Queries) error {
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
