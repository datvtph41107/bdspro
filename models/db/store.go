package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transactioner is the sole transaction capability shared by account,
// authentication and organization model operations.
type Transactioner interface {
	ExecTx(context.Context, func(*Queries) error) error
}

// Store exposes SQLC queries and the existing transaction operations.
type Store interface {
	Querier
	GetAccountByID(ctx context.Context, id int64) (Account, error)
	GetAccountByEmail(ctx context.Context, email string) (Account, error)
	Transactioner
	// Cross-protocol recovery needs one PostgreSQL transaction for SQLC
	// credential/session writes and OAuth grant invalidation.
	ExecTxRaw(context.Context, func(pgx.Tx) error) error
}

type SQLStore struct {
	pool *pgxpool.Pool
	*Queries
}

func NewStore(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{pool: pool, Queries: New(contextDBTX{pool: pool})}
}

func (s *SQLStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

var _ Store = (*SQLStore)(nil)
