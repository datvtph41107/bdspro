package db

import (
	"context"
	"sync"
	"time"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/jackc/pgx/v5"
)

type transactionContextKey struct{}

// A scope belongs to one synchronous transaction workflow. The mutex protects
// state transitions, not concurrent use of pgx.Tx or SQLC queries.
type transactionScope struct {
	tx      pgx.Tx
	queries *sqlc.Queries

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
	if s.err == nil && (s.tx == nil || s.queries == nil) {
		s.err = ErrTransactionContext
	}
	return s.err
}

func (s *transactionScope) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

// InTransaction reports a usable transaction scope, not merely a context value.
func InTransaction(ctx context.Context) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	scope, ok := ctx.Value(transactionContextKey{}).(*transactionScope)
	return ok && scope != nil && scope.failure() == nil
}

// WithTx owns begin/commit/rollback at the outermost call. Nested calls join
// the same transaction; a nested failure poisons the entire scope. No savepoint
// is created. Callbacks must not use the transaction concurrently or retain it.
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
	if fn == nil {
		return ErrTransactionCallback
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

	// Rollback is safe after commit; pgx releases the acquired pool connection.
	// Cleanup must not inherit the canceled request context.
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
	if fn == nil {
		scope.fail(ErrTransactionCallback)
		return ErrTransactionCallback
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

// WithTx2 returns the callback result only when WithTx succeeds. A nested
// result remains provisional until the outermost transaction commits.
func WithTx2[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var value T
	if fn == nil {
		return value, WithTx(ctx, nil)
	}
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
