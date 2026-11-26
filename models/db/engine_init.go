package db

import (
	"context"
	"sync"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/setting"
)

var defaultEngine struct {
	mu     sync.RWMutex
	value  *engine
	closed bool
}

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

	defaultEngine.value = &engine{
		pool:    pool,
		queries: New(pool),
	}

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
