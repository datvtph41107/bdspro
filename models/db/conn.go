package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a PostgreSQL connection pool, verifies it, and closes on
// startup failure. No transaction or domain behavior belongs in this file.
func Connect(ctx context.Context, source string) (*pgxpool.Pool, error) {
	if source == "" {
		return nil, fmt.Errorf("PostgreSQL source is required")
	}
	pool, err := pgxpool.New(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return pool, nil
}
