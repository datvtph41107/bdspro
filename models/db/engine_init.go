package db

import (
	"context"
	"errors"
	"sync"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// InitStore is retained for callers that own an independent database pool,
// such as migrations, cleanup commands, and isolated test fixtures.
func InitStore(ctx context.Context, source string) (*SQLStore, error) {
	pool, err := Connect(ctx, source)
	if err != nil {
		return nil, err
	}
	return NewStore(pool), nil
}

var (
	ErrEngineAlreadyInitialized = errors.New("db: engine already initialized")
	ErrEngineNotInitialized     = errors.New("db: engine not initialized")
	ErrEngineClosed             = errors.New("db: engine closed")
)

var defaultEngine struct {
	mu     sync.RWMutex
	store  *SQLStore
	closed bool
}

// InitEngine initializes the process-owned SQLC store exactly once.
// The returned store is borrowed; subsystem consumers must not close it.
// The graceful manager closes it after serving activities have drained.
func InitEngine(ctx context.Context) (*SQLStore, error) {
	defaultEngine.mu.Lock()
	defer defaultEngine.mu.Unlock()

	if defaultEngine.closed {
		return nil, ErrEngineClosed
	}
	if defaultEngine.store != nil {
		return nil, ErrEngineAlreadyInitialized
	}

	store, err := InitStore(ctx, setting.Database.DBSource)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		store.Close()
		return nil, err
	}

	// Cleanup registration is part of initialization success, not a
	// responsibility returned to routers/init.go or routers.Server.
	graceful.GetManager().RunAtTerminate(CloseEngine)
	defaultEngine.store = store
	return store, nil
}

// ProcessStore returns the initialized borrowed store for subsystem setup.
// Domain SQLC queries must continue to use context-aware transaction APIs.
func ProcessStore() (Store, error) {
	defaultEngine.mu.RLock()
	defer defaultEngine.mu.RUnlock()
	if defaultEngine.closed {
		return nil, ErrEngineClosed
	}
	if defaultEngine.store == nil {
		return nil, ErrEngineNotInitialized
	}
	return defaultEngine.store, nil
}

// CloseEngine is terminal and idempotent for this process.
// The graceful terminate phase, after active operations have drained,
// is the only production caller.
func CloseEngine() {
	defaultEngine.mu.Lock()
	store := defaultEngine.store
	defaultEngine.store = nil
	defaultEngine.closed = true
	defaultEngine.mu.Unlock()

	if store != nil {
		store.Close()
	}
}
