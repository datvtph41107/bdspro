package db

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
)

type storeContextKey struct{}
type queriesContextKey struct{}
type txStateContextKey struct{}
type rawTxContextKey struct{}

// A recovered panic in a nested operation must not allow its parent to commit.
// The original panic is re-thrown; this error is only the rollback-only cause.
var errTransactionPanicked = errors.New("db: nested transaction callback panicked")

// transactionState ensures an inner transaction error poisons the outer commit
// even if its caller ignores the returned error. It is request-local.
type transactionState struct {
	mu     sync.Mutex
	err    error
	closed bool
}

func (s *transactionState) fail(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

func (s *transactionState) failure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrTransactionClosed
	}
	return s.err
}

func (s *transactionState) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

var (
	ErrStoreMissing         = errors.New("db: store missing from context")
	ErrTxStoreUnavailable   = errors.New("db: transaction-bound store unavailable")
	ErrTransactionRequired  = errors.New("db: transaction required")
	ErrTransactionClosed    = errors.New("db: transaction context is no longer active")
	ErrNestedTransaction    = errors.New("db: nested transaction context has no owner")
	ErrRawTransactionInSQLC = errors.New("db: raw transaction cannot nest inside SQLC transaction")
)

// WithStore installs the transaction capability in the request context. The
// ingress normally supplies a full Store; model operations which only require
// transactions may supply their existing narrow Transactioner. Reads that need
// a full Store continue through StoreFromContext and cannot use a tx-only fake.
// No process-global mutable database dependency is introduced.
func WithStore(ctx context.Context, store Transactioner) context.Context {
	if store == nil {
		panic("db: nil store")
	}
	return context.WithValue(ctx, storeContextKey{}, store)
}

// StoreFromContext resolves a query capability for the current context.
// Outside a transaction it returns the ingress Store. Inside a transaction
// it MUST expose only the transaction's SQLC executor, matching Gitea's
// context-aware GetEngine behavior. A pool-backed Store would silently allow
// domain reads/writes to escape the caller's rollback boundary.
//
// A narrow transaction-only mock has no complete query Store to rebind; fail
// closed instead of silently returning an unrelated pool-backed executor.
func StoreFromContext(ctx context.Context) (Store, error) {
	if ctx.Value(queriesContextKey{}) != nil {
		q, err := TxQueries(ctx)
		if err != nil {
			return nil, err
		}
		store, ok := ctx.Value(storeContextKey{}).(*SQLStore)
		if !ok || store == nil {
			if state, ok := ctx.Value(txStateContextKey{}).(*transactionState); ok {
				state.fail(ErrTxStoreUnavailable)
			}
			return nil, ErrTxStoreUnavailable
		}
		// Do not carry the pool into this transaction-scoped view. Promoted
		// SQLC methods now use q, and its ExecTx/ExecTxRaw cannot create
		// an independent transaction when the context is lost.
		return &SQLStore{Queries: q}, nil
	}
	store, ok := ctx.Value(storeContextKey{}).(Store)
	if !ok || store == nil {
		return nil, ErrStoreMissing
	}
	return store, nil
}

// TxQueries retrieves the single SQLC executor belonging to the transaction.
// It must never silently fall back to non-transactional queries.
func TxQueries(ctx context.Context) (*Queries, error) {
	q, ok := ctx.Value(queriesContextKey{}).(*Queries)
	if !ok || q == nil {
		return nil, ErrTransactionRequired
	}
	state, ok := ctx.Value(txStateContextKey{}).(*transactionState)
	if !ok || state == nil {
		return nil, ErrNestedTransaction
	}
	if err := state.failure(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		state.fail(err)
		return nil, err
	}
	return q, nil
}

// WithTx matches Gitea's context-first transaction pattern: reuse the
// current executor when a transaction already exists; otherwise open exactly
// one transaction from the Store installed at ingress.
//
// An inner callback error marks the transaction rollback-only. The outermost
// call cannot commit it even when an intermediate caller discards that error.
func WithTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(queriesContextKey{}) != nil {
		state, ok := ctx.Value(txStateContextKey{}).(*transactionState)
		if !ok || state == nil {
			return ErrNestedTransaction
		}
		if err := state.failure(); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			state.fail(err)
			return err
		}
		// A caller may recover an inner panic and accidentally return nil from
		// the outer callback. Preserve the panic AND poison the shared owner.
		defer func() {
			if recovered := recover(); recovered != nil {
				state.fail(errTransactionPanicked)
				panic(recovered)
			}
		}()
		err := fn(ctx)
		state.fail(err)
		state.fail(ctx.Err())
		if err != nil {
			return err
		}
		return state.failure()
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	store, ok := ctx.Value(storeContextKey{}).(Transactioner)
	if !ok || store == nil {
		return ErrStoreMissing
	}
	state := &transactionState{}
	// The SQLStore implementation exposes the pgx.Tx to the context-aware
	// SQLC executor, so even an ingress Store passed directly to a model uses
	// the active transaction when the model forwards txCtx. The same
	// ExecTxRaw implementation owns Begin/Commit/Rollback as before.
	run := func(parentCtx context.Context, q *Queries) error {
		defer state.close()
		txCtx := context.WithValue(parentCtx, queriesContextKey{}, q)
		txCtx = context.WithValue(txCtx, txStateContextKey{}, state)
		err := fn(txCtx)
		state.fail(err)
		state.fail(txCtx.Err())
		return state.failure()
	}
	if sqlStore, ok := store.(*SQLStore); ok {
		return sqlStore.ExecTxRaw(ctx, func(tx pgx.Tx) error {
			txCtx := context.WithValue(ctx, rawTxContextKey{}, tx)
			return run(txCtx, New(tx))
		})
	}
	return store.ExecTx(ctx, func(q *Queries) error {
		return run(ctx, q)
	})
}

// WithTx2 mirrors Gitea's result-bearing context transaction entrypoint.
// When it owns the outermost transaction, a failed callback or COMMIT returns
// the zero value. Under an existing transaction its result is PROVISIONAL:
// only the parent WithTx can commit it. Do not expose such a result externally
// before that parent returns successfully.
func WithTx2[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var result T
	err := WithTx(ctx, func(txCtx context.Context) error {
		next, err := fn(txCtx)
		if err != nil {
			return err
		}
		result = next
		return nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}
