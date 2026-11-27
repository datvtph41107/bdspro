package db

import (
	"context"
	"errors"
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
	if ctx == nil {
		return errors.New("db: nil initialization context")
	}
	defaultEngine.mu.Lock()
	defer defaultEngine.mu.Unlock()

	if defaultEngine.closed {
		return ErrEngineClosed
	}

	if defaultEngine.value != nil {
		return ErrEngineAlreadyInitialized
	}

	manager := graceful.GetManager()
	if manager == nil {
		return errors.New("db: graceful manager not initialized")
	}

	pool, err := openPool(ctx, setting.Database.DBSource, &queryLogger{mode: setting.Database.LogSQL, slow: setting.Database.SlowQueryThreshold})
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		pool.Close()
		return err
	}

	manager.RunAtTerminate(CloseEngine)

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
