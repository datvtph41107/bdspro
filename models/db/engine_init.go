package db

import "context"

// InitStore remains the legacy constructor until domain Store users are
// migrated to the canonical context-based database access contract.
// InitStore owns connection setup for a process. Call store.Close on shutdown.
func InitStore(ctx context.Context, source string) (*SQLStore, error) {
	pool, err := Connect(ctx, source)
	if err != nil {
		return nil, err
	}
	return NewStore(pool), nil
}
