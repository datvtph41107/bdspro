package conn

import (
	"context"
	"errors"
	"sync"

	redis "github.com/go-redis/redis/v8"
)

var (
	ErrRedisAlreadyInitialized = errors.New(
		"conn: Redis already initialized",
	)
	ErrRedisClosed = errors.New(
		"conn: Redis connection closed",
	)
)

var processRedis struct {
	mu      sync.Mutex
	manager *RedisManager
	client  *redis.Client
	closed  bool
}

// InitRedis creates the process-owned Redis client.
// Callers borrow the client; they do not own its cleanup. The application
// composition root supplies the address and registers termination cleanup.
func InitRedis(ctx context.Context, address string) (*redis.Client, error) {
	processRedis.mu.Lock()
	defer processRedis.mu.Unlock()

	if processRedis.closed {
		return nil, ErrRedisClosed
	}

	if processRedis.manager != nil {
		return nil, ErrRedisAlreadyInitialized
	}

	manager := NewRedisManager()

	client, err := manager.Client(ctx, address)
	if err != nil {
		_ = manager.Close()
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		_ = manager.Close()
		return nil, err
	}

	processRedis.manager = manager
	processRedis.client = client

	return client, nil
}

// ProcessRedis returns the shared, borrowed client.
func ProcessRedis() (*redis.Client, error) {
	processRedis.mu.Lock()
	defer processRedis.mu.Unlock()

	if processRedis.closed {
		return nil, ErrRedisClosed
	}

	if processRedis.client == nil {
		return nil, errors.New(
			"conn: Redis not initialized",
		)
	}

	return processRedis.client, nil
}

// CloseRedis releases process-owned Redis connections.
// Repeated calls are safe.
func CloseRedis() {
	processRedis.mu.Lock()

	manager := processRedis.manager

	processRedis.manager = nil
	processRedis.client = nil
	processRedis.closed = true

	processRedis.mu.Unlock()

	if manager != nil {
		_ = manager.Close()
	}
}
