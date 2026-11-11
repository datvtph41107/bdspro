package conn

import (
	"context"
	"errors"
	"sync"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/setting"
	redis "github.com/go-redis/redis/v8"
)

var (
	ErrRedisAlreadyInitialized = errors.New("conn: Redis already initialized")
	ErrRedisClosed             = errors.New("conn: Redis connection closed")
)

var processRedis struct {
	mu      sync.Mutex
	manager *RedisManager
	client  *redis.Client
	closed  bool
}

// InitRedis initializes the process Redis client and registers its cleanup.
// The returned client is borrowed: only CloseRedis owns its lifetime.
func InitRedis(ctx context.Context) (*redis.Client, error) {
	processRedis.mu.Lock()
	defer processRedis.mu.Unlock()

	if processRedis.closed {
		return nil, ErrRedisClosed
	}
	if processRedis.manager != nil {
		return nil, ErrRedisAlreadyInitialized
	}

	manager := NewRedisManager()
	client, err := manager.Client(ctx, setting.Redis.Address)
	if err != nil {
		_ = manager.Close()
		return nil, err
	}

	graceful.GetManager().RunAtTerminate(CloseRedis)
	processRedis.manager = manager
	processRedis.client = client
	return client, nil
}

// ProcessRedis returns the validated process-owned Redis client.
func ProcessRedis() (*redis.Client, error) {
	processRedis.mu.Lock()
	defer processRedis.mu.Unlock()
	if processRedis.closed {
		return nil, ErrRedisClosed
	}
	if processRedis.manager == nil {
		return nil, errors.New("conn: Redis not initialized")
	}
	return processRedis.client, nil
}

// CloseRedis closes only process-owned clients, after graceful drain.
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
