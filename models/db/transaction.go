package db

import "context"

// WithTxResult executes one SQLC transaction and publishes a result only after
// ExecTx has successfully committed. A rollback or failed commit always returns
// T's zero value, even when the callback produced a non-zero result.
//
// Pass the transaction-scoped *Queries into shared persistence operations.
// Do not call ExecTx on the outer store from inside this callback: that would
// open a second, independently committed transaction.
func WithTxResult[T any](ctx context.Context, tx Transactioner, fn func(*Queries) (T, error)) (T, error) {
	var result T
	err := tx.ExecTx(ctx, func(q *Queries) error {
		next, err := fn(q)
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
