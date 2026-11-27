package db

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type transactionContextKey struct{}

type transactionScope struct {
	tx      pgx.Tx
	queries *Queries

	mu     sync.Mutex
	err    error
	closed bool
}

func (s *transactionScope) fail(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

func (s *transactionScope) failure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrTransactionClosed
	}
	return s.err
}

func (s *transactionScope) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

// GetEngine returns the current SQL execution handle for legacy inline SQL.
// Prefer GetQueries for generated SQLC queries.
func GetEngine(ctx context.Context) (DBTX, error) {
	conn, _, err := resolveQueryContext(ctx)
	return conn, err
}

func GetQueries(ctx context.Context) (*Queries, error) {
	_, queries, err := resolveQueryContext(ctx)
	return queries, err
}

// Both handles must refer to the same pool or the same transaction.
// An invalid transaction context must never fall back to the default pool.
func resolveQueryContext(ctx context.Context) (DBTX, *Queries, error) {
	if ctx == nil {
		return nil, nil, ErrTransactionContext
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if value := ctx.Value(transactionContextKey{}); value != nil {
		scope, ok := value.(*transactionScope)
		if !ok || scope == nil {
			return nil, nil, ErrTransactionContext
		}
		if err := scope.failure(); err != nil {
			return nil, nil, err
		}
		if scope.tx == nil || scope.queries == nil {
			scope.fail(ErrTransactionContext)
			return nil, nil, ErrTransactionContext
		}
		return scope.tx, scope.queries, nil
	}
	current, err := getDefaultEngine()
	if err != nil {
		return nil, nil, err
	}
	return current.pool, current.queries, nil
}

func InTransaction(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	scope, ok := ctx.Value(transactionContextKey{}).(*transactionScope)
	return ok && scope != nil && scope.failure() == nil
}

func WithTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		return ErrTransactionContext
	}
	if value := ctx.Value(transactionContextKey{}); value != nil {
		scope, ok := value.(*transactionScope)
		if !ok || scope == nil {
			return ErrTransactionContext
		}
		return reuseTransaction(ctx, scope, fn)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := getDefaultEngine()
	if err != nil {
		return err
	}
	tx, err := current.pool.Begin(ctx)
	if err != nil {
		return err
	}
	scope := &transactionScope{tx: tx}
	scope.queries = current.queries.WithTx(tx)
	txCtx := context.WithValue(ctx, transactionContextKey{}, scope)

	defer func() {
		scope.close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	if err := fn(txCtx); err != nil {
		scope.fail(err)
		return err
	}
	if err := ctx.Err(); err != nil {
		scope.fail(err)
		return err
	}
	if err := scope.failure(); err != nil {
		return err
	}
	scope.close()
	return tx.Commit(ctx)
}

func reuseTransaction(ctx context.Context, scope *transactionScope, fn func(context.Context) error) (err error) {
	if err := scope.failure(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		scope.fail(err)
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			scope.fail(errTransactionPanicked)
			panic(recovered)
		}
		scope.fail(err)
		scope.fail(ctx.Err())
		if err == nil {
			err = scope.failure()
		}
	}()
	return fn(ctx)
}

func WithTx2[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var value T
	err := WithTx(ctx, func(txCtx context.Context) error {
		next, err := fn(txCtx)
		if err == nil {
			value = next
		}
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}
