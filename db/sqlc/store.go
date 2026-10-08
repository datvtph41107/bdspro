package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store exposes generated queries together with transaction execution.
type Store interface {
	Querier

	ExecTx(
		ctx context.Context,
		fn func(Querier) error,
	) error
}

// SQLStore implements Store with PostgreSQL via pgxpool.
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

// NewStore creates a Store backed by one shared pgx pool.
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
