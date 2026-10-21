package db

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
    Querier
    RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error)
}

type SQLStore struct {
    pool *pgxpool.Pool
    *Queries
}

func NewStore(pool *pgxpool.Pool) Store {
    return &SQLStore{pool: pool, Queries: New(pool)}
}

type RegisterTxParams struct {
    Email        string
    PasswordHash string
}

func (s *SQLStore) RegisterTx(ctx context.Context, arg RegisterTxParams) (Account, error) {
    tx, err := s.pool.Begin(ctx)
    if err != nil { return Account{}, err }
    defer tx.Rollback(ctx)
    q := New(tx)
    partyID, err := q.CreatePersonParty(ctx)
    if err != nil { return Account{}, err }
    if err := q.CreatePerson(ctx, partyID); err != nil { return Account{}, err }
    account, err := q.CreateAccount(ctx, CreateAccountParams{
        PersonID: partyID, Email: arg.Email, PasswordHash: arg.PasswordHash,
    })
    if err != nil { return Account{}, err }
    if err := tx.Commit(ctx); err != nil { return Account{}, err }
    return account, nil
}

var _ Store = (*SQLStore)(nil)
