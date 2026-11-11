package db

import "context"

// WithTxResult is retained for legacy model/test consumers. New code must use
// WithTx2 and GetQueries(ctx). Transaction ownership is exclusively delegated
// to WithTx: this compatibility entrypoint cannot open its own transaction.
//
// Deprecated: migrate callers to WithTx2 with a transaction-scoped context.
func WithTxResult[T any](ctx context.Context, tx Transactioner, fn func(*Queries) (T, error)) (T, error) {
	if !hasTransactionMetadata(ctx) {
		ctx = WithStore(ctx, tx)
	}
	return WithTx2(ctx, func(txCtx context.Context) (T, error) {
		q, err := TxQueries(txCtx)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(q)
	})
}
