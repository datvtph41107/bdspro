package db

import "context"

// WithTxResult runs a SQLC transaction and returns its result only after
// the transaction owner commits. When ctx already carries a transaction,
// it reuses that executor; the result is then provisional until the outer
// transaction commits. A nested failure poisons that outer transaction.
//
// tx is the standalone transaction capability. It must not open a second
// transaction when the caller supplies a transaction-scoped context.
func WithTxResult[T any](ctx context.Context, tx Transactioner, fn func(*Queries) (T, error)) (T, error) {
	var result T
	if err := ctx.Err(); err != nil {
		return result, err
	}

	run := func(q *Queries) error {
		next, err := fn(q)
		if err != nil {
			return err
		}
		result = next
		return ctx.Err()
	}

	var err error
	if ctx.Value(queriesContextKey{}) != nil {
		err = WithTx(ctx, func(txCtx context.Context) error {
			q, err := TxQueries(txCtx)
			if err != nil {
				return err
			}
			return run(q)
		})
	} else {
		err = tx.ExecTx(ctx, run)
	}
	if err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}
