package db

import (
	"context"
	"errors"
	"sync"
)

type storeContextKey struct{}
type queriesContextKey struct{}
type txStateContextKey struct{}

// transactionState ensures an inner transaction error poisons the outer commit
// even if its caller ignores the returned error. It is request-local.
type transactionState struct {
	mu  sync.Mutex
	err error
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
	return s.err
}

var (
	ErrStoreMissing        = errors.New("db: store missing from context")
	ErrTransactionRequired = errors.New("db: transaction required")
	ErrNestedTransaction   = errors.New("db: nested transaction context has no owner")
)

// WithStore is called by the request/worker/test entrypoint, not by each model
// operation. The Store remains request-scoped, never a mutable global.
func WithStore(ctx context.Context, store Store) context.Context {
	if store == nil {
		panic("db: nil store")
	}
	return context.WithValue(ctx, storeContextKey{}, store)
}

func StoreFromContext(ctx context.Context) (Store, error) {
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
		err := fn(ctx)
		state.fail(err)
		if err != nil {
			return err
		}
		return state.failure()
	}

	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}
	state := &transactionState{}
	return store.ExecTx(ctx, func(q *Queries) error {
		txCtx := context.WithValue(ctx, queriesContextKey{}, q)
		txCtx = context.WithValue(txCtx, txStateContextKey{}, state)
		err := fn(txCtx)
		state.fail(err)
		return state.failure()
	})
}

// WithTx2 mirrors Gitea's named context+result transaction entrypoint.
// Unlike Gitea's simple WithTx2 implementation, a failed COMMIT discards
// the callback's result so a non-persisted SQLC row cannot escape.
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
