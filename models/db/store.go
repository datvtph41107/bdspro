package db

import (
	"context"
	"time"

	sqlc "github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	return &SQLStore{pool: pool, Queries: New(contextDBTX{pool: pool})}
}

func (s *SQLStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// contextDBTX is a DBTX executor selector, not a second transaction owner.
// It is installed only on an ingress SQLStore. In a transaction callback,
// the same pgx.Tx opened by SQLStore.ExecTxRaw is selected from txCtx;
// ordinary requests continue to use the pool. SQLC-generated code stays
// unchanged and both paths retain the same DBTX method signatures.
type contextDBTX struct {
	pool *pgxpool.Pool
}

func (d contextDBTX) executor(ctx context.Context) (DBTX, error) {
	if ctx.Value(queriesContextKey{}) == nil {
		return d.pool, nil
	}
	if _, err := TxQueries(ctx); err != nil {
		return nil, err
	}
	// A context from SQLStore A must not redirect SQLStore B to A's pool.
	// Bind by pool identity so two stores for the same pool can interoperate,
	// while a different database/tenant fails closed.
	if ctx.Value(rawTxStoreContextKey{}) == nil || ctx.Value(rawTxStoreContextKey{}) != d.pool {
		if state, ok := ctx.Value(txStateContextKey{}).(*transactionState); ok {
			state.fail(ErrTxStoreUnavailable)
		}
		return nil, ErrTxStoreUnavailable
	}
	tx, ok := ctx.Value(rawTxContextKey{}).(pgx.Tx)
	if !ok || tx == nil {
		if state, ok := ctx.Value(txStateContextKey{}).(*transactionState); ok {
			state.fail(ErrTxStoreUnavailable)
		}
		return nil, ErrTxStoreUnavailable
	}
	return tx, nil
}

func (d contextDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return exec.Exec(ctx, sql, args...)
}

func (d contextDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	exec, err := d.executor(ctx)
	if err != nil {
		return nil, err
	}
	return exec.Query(ctx, sql, args...)
}

type queryErrorRow struct{ err error }

func (r queryErrorRow) Scan(...any) error { return r.err }

func (d contextDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	exec, err := d.executor(ctx)
	if err != nil {
		return queryErrorRow{err: err}
	}
	return exec.QueryRow(ctx, sql, args...)
}

var _ DBTX = contextDBTX{}

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
	if s.pool == nil {
		return ErrTransactionClosed
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
	if s.pool == nil {
		return ErrTransactionClosed
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
