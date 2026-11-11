package db

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type transactionContextKey struct{}

// transactionScope is the sole context authority for PostgreSQL transactions.
// Its mutex protects lifecycle state, not concurrent use of pgx.Tx.
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

// GetEngine returns the same SQL execution authority selected by context.
// It mirrors Gitea's GetEngine(ctx), but uses SQLC's DBTX interface instead
// of XORM. It never creates a second transaction or a database Store.
func GetEngine(ctx context.Context) (DBTX, error) {
	if ctx == nil {
		return nil, ErrTransactionContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if value := ctx.Value(transactionContextKey{}); value != nil {
		scope, ok := value.(*transactionScope)
		if !ok || scope == nil {
			return nil, ErrTransactionContext
		}
		if err := scope.failure(); err != nil {
			return nil, err
		}
		return transactionDBTX{scope: scope}, nil
	}
	current, err := getDefaultEngine()
	if err != nil {
		return nil, err
	}
	return poolDBTX{pool: current.pool}, nil
}

// GetQueries returns the transaction's SQLC executor if a transaction exists,
// otherwise the process engine's executor. No Store or adapter is injected at
// the HTTP boundary.
func GetQueries(ctx context.Context) (*Queries, error) {
	if ctx == nil {
		return nil, ErrTransactionContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if value := ctx.Value(transactionContextKey{}); value != nil {
		scope, ok := value.(*transactionScope)
		if !ok || scope == nil {
			return nil, ErrTransactionContext
		}
		if err := scope.failure(); err != nil {
			return nil, err
		}
		if scope.queries == nil {
			scope.fail(ErrTransactionContext)
			return nil, ErrTransactionContext
		}
		return scope.queries, nil
	}
	current, err := getDefaultEngine()
	if err != nil {
		return nil, err
	}
	return current.queries, nil
}

// InTransaction matches Gitea's context-based transaction inspection.
func InTransaction(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	scope, ok := ctx.Value(transactionContextKey{}).(*transactionScope)
	return ok && scope != nil && scope.failure() == nil
}

// WithTx owns Begin/Commit/Rollback. Nested calls reuse the same transaction;
// an ignored nested failure marks the outer transaction rollback-only.
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
	scope.queries = New(transactionDBTX{scope: scope})
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

// WithTx2 is Gitea's result-bearing form of WithTx. A nested result is
// provisional until its outermost transaction commits.
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
