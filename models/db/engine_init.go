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

// InitEngine creates and publishes the process-owned
// PostgreSQL engine.
//
// It is called once during application startup,
// after Common settings and the graceful Manager exist.
func InitEngine(ctx context.Context) error {
	defaultEngine.mu.Lock()
	defer defaultEngine.mu.Unlock()

	if defaultEngine.closed {
		return ErrEngineClosed
	}

	if defaultEngine.value != nil {
		return ErrEngineAlreadyInitialized
	}

	pool, err := Connect(
		ctx,
		setting.Database.DBSource,
	)
	if err != nil {
		return err
	}

	// A connection established during cancellation
	// must not become a published process resource.
	if err := ctx.Err(); err != nil {
		pool.Close()
		return err
	}

	// Publication occurs only after the resource
	// has been verified and cleanup is registered.
	//
	// The startup activity registration in cmd
	// prevents termination before initialization ends.
	graceful.GetManager().RunAtTerminate(CloseEngine)

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
