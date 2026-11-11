package db

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/setting"
)

type engine struct {
	pool    *pgxpool.Pool
	queries *Queries
}

var defaultEngine struct {
	mu     sync.RWMutex
	value  *engine
	closed bool
}

// InitEngine publishes exactly one process-owned PostgreSQL engine.
// It must be called after settings and graceful manager initialization.
func InitEngine(ctx context.Context) error {
	defaultEngine.mu.Lock()
	defer defaultEngine.mu.Unlock()
	if defaultEngine.closed {
		return ErrEngineClosed
	}
	if defaultEngine.value != nil {
		return ErrEngineAlreadyInitialized
	}
	pool, err := Connect(ctx, setting.Database.DBSource)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		pool.Close()
		return err
	}
	graceful.GetManager().RunAtTerminate(CloseEngine)
	defaultEngine.value = &engine{pool: pool, queries: New(poolDBTX{pool: pool})}
	return nil
}

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

// CloseEngine is a terminal process shutdown operation. Call only after
// serving activities have drained.
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
