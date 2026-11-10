package db

import (
	"context"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
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

// New exposes SQLC query construction through the stable db package.
func New(conn DBTX) *Queries {
	return sqlc.New(conn)
}

func NewStore(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{pool: pool, Queries: New(pool)}
}

func (s *SQLStore) Close() {
	s.pool.Close()
}

// ExecTx owns transaction creation, commit and rollback.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	// A transaction-scoped context owns the executor. Never acquire a second
	// pool connection or commit independently of its outer transaction.
	if ctx.Value(queriesContextKey{}) != nil {
		return WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			return fn(q)
		})
	}

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
