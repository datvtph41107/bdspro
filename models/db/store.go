package db

import (
	"context"
	"time"

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

// queryExecutor is the single raw-SQL read/write selector for SQLStore
// methods that do not own a transaction themselves (e.g. OAuth client reads).
// Its contract mirrors Gitea GetEngine(ctx): operations with txCtx must use
// the existing owner, not a separate pool connection. Generated SQLC methods
// use contextDBTX directly; raw SQL methods use this same selector.
func (s *SQLStore) queryExecutor(ctx context.Context) (DBTX, error) {
	if s.pool != nil {
		return (contextDBTX{pool: s.pool}).executor(ctx)
	}
	if ctx.Value(queriesContextKey{}) == nil {
		return nil, ErrTransactionClosed
	}
	q, err := TxQueries(ctx)
	if err != nil {
		return nil, err
	}
	if s.Queries != q {
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
			// Context is the transaction authority, but the explicit receiver
			// must belong to that authority. Otherwise store B could silently
			// execute against store A's transaction. The callback error poisons
			// the outer owner even if an intermediate caller ignores it.
			if txCtx.Value(rawTxStoreContextKey{}) != nil && !transactionStoreMatches(txCtx, s, q) {
				return ErrTxStoreUnavailable
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
