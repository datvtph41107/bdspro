package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, source string) (*pgxpool.Pool, error) {
	return connect(ctx, source, nil)
}

func connect(ctx context.Context, source string, tracer pgx.QueryTracer) (*pgxpool.Pool, error) {
	if source == "" {
		return nil, fmt.Errorf("PostgreSQL source is required")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration")
	}
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return pool, nil
}
