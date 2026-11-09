package db

import (
	"context"
	"errors"
)

type storeContextKey struct{}
type queriesContextKey struct{}

var (
	ErrStoreMissing        = errors.New("db: store missing from context")
	ErrTransactionRequired = errors.New("db: transaction required")
	ErrNestedTransaction   = errors.New("db: nested transaction is not supported")
)

// Store is installed by the request/job entrypoint; no mutable global engine.
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

// TxQueries never falls back to a non-transactional executor during writes.
func TxQueries(ctx context.Context) (*Queries, error) {
	q, ok := ctx.Value(queriesContextKey{}).(*Queries)
	if !ok || q == nil {
		return nil, ErrTransactionRequired
	}
	return q, nil
}

// WithTx rejects nested transactions until rollback-only propagation is tested.
func WithTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(queriesContextKey{}) != nil {
		return ErrNestedTransaction
	}
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}
	return store.ExecTx(ctx, func(q *Queries) error {
		return fn(context.WithValue(ctx, queriesContextKey{}, q))
	})
}

// WithTxValue publishes a persisted result only after the outer commit succeeds.
func WithTxValue[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
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
