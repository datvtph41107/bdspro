package db

import "context"

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
		return scope.tx, scope.queries, nil
	}
	current, err := getDefaultEngine()
	if err != nil {
		return nil, nil, err
	}
	return current.pool, current.queries, nil
}
