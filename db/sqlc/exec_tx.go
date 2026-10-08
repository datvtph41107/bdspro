package db

import (
	"context"
	"fmt"
)

// ExecTx executes fn inside one PostgreSQL transaction.
func (store *SQLStore) ExecTx(
	ctx context.Context,
	fn func(Querier) error,
) error {
	tx, err := store.connPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin transaction: %w",
			err,
		)
	}

	queries := store.Queries.WithTx(tx)

	if err := fn(queries); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf(
				"transaction error: %w; rollback error: %v",
				err,
				rollbackErr,
			)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit transaction: %w",
			err,
		)
	}

	return nil
}
