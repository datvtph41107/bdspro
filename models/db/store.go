package db

import (
	"context"
	"time"

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

// ExecTx runs SQLC mutations inside the owning PostgreSQL transaction.
// An existing context-scoped SQLC executor always takes precedence over the
// Store's pool; nested calls may not commit independently.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	if ctx.Value(queriesContextKey{}) != nil {
		return WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			return fn(q)
		})
	}
	return s.ExecTxRaw(ctx, func(tx pgx.Tx) error {
		return fn(New(tx))
	})
}

// ExecTxRaw is the single PostgreSQL Begin/Commit/Rollback owner for SQLStore.
// OAuth and credential recovery need raw pgx.Tx for operations which SQLC does
// not generate. Raw nesting inside an active SQLC transaction is forbidden:
// opening another pool connection would permit an independent commit.
func (s *SQLStore) ExecTxRaw(ctx context.Context, fn func(pgx.Tx) error) error {
	if ctx.Value(queriesContextKey{}) != nil {
		// Poison the parent so swallowing the error cannot commit it.
		return WithTx(ctx, func(context.Context) error { return ErrRawTransactionInSQLC })
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	// PostgreSQL cleanup must not inherit a canceled request context.
	// The transaction owner still decides COMMIT using the original ctx;
	// rollback uses an independent bounded context to release the connection.
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ Store = (*SQLStore)(nil)
