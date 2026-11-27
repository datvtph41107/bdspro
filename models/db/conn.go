package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxpool may continue establishing connections after an acquire is canceled.
// Bound both the driver's connection attempt and the initial readiness check,
// unless the deployment supplies its own positive connect_timeout.
const defaultConnectTimeout = 5 * time.Second

func openPool(ctx context.Context, source string, tracer pgx.QueryTracer) (*pgxpool.Pool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == "" {
		return nil, errors.New("PostgreSQL source is required")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, errors.New("invalid PostgreSQL configuration")
	}
	if cfg.ConnConfig.ConnectTimeout <= 0 {
		cfg.ConnConfig.ConnectTimeout = defaultConnectTimeout
	}
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	readyCtx, cancel := context.WithTimeout(ctx, cfg.ConnConfig.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(readyCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return pool, nil
}
