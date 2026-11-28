package db

import (
	"context"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
)

// GetQueries returns the generated query object bound to the current scope.
// An invalid transaction context must never fall back to the default pool.
func GetQueries(ctx context.Context) (*sqlc.Queries, error) {
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
		return scope.queries, nil
	}
	current, err := getDefaultEngine()
	if err != nil {
		return nil, err
	}
	return current.queries, nil
}
