package db

import (
	"context"
	"sync"

)

var defaultEngine struct {
	mu     sync.RWMutex
	value  *engine
	closed bool
}

// InitEngine creates and publishes the process-owned
// PostgreSQL engine.
//
// The process composition root supplies the connection string and registers
// CloseEngine at termination. This package does not control process lifecycle.
func InitEngine(ctx context.Context, dsn string) error {
	defaultEngine.mu.Lock()
	defer defaultEngine.mu.Unlock()

	if defaultEngine.closed {
		return ErrEngineClosed
	}

	if defaultEngine.value != nil {
		return ErrEngineAlreadyInitialized
	}

	pool, err := Connect(ctx, dsn)
	if err != nil {
		return err
	}

	// A connection established during cancellation
	// must not become a published process resource.
	if err := ctx.Err(); err != nil {
		pool.Close()
		return err
	}

	// The caller owns lifecycle registration before making the application
	// available to consumers; this package only publishes the verified pool.
	defaultEngine.value = &engine{
		pool:    pool,
		queries: New(pool),
	}

	return nil
}

// getDefaultEngine returns the active process engine.
func getDefaultEngine() (*engine, error) {
	defaultEngine.mu.RLock()
	defer defaultEngine.mu.RUnlock()

	if defaultEngine.closed {
		return nil, ErrEngineClosed
	}

	if defaultEngine.value == nil {
		return nil, ErrEngineNotInitialized
	}

	return defaultEngine.value, nil
}

// CloseEngine releases the process PostgreSQL pool.
// It is idempotent across startup rollback and termination.
func CloseEngine() {
	defaultEngine.mu.Lock()

	current := defaultEngine.value
	defaultEngine.value = nil
	defaultEngine.closed = true

	defaultEngine.mu.Unlock()

	if current != nil {
		current.pool.Close()
	}
}
