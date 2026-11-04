// Package kvstore owns Redis client connectivity shared by key-value consumers.
// Cache and security state use the client; Asynq owns its own Redis connections.
package kvstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
)

// RedisManager owns process-scoped clients. Only the manager closes them;
// callers must not close a client borrowed from Client.
type RedisManager struct {
	mu      sync.Mutex
	clients map[string]*redis.Client
	closed  bool
}

func NewRedisManager() *RedisManager {
	return &RedisManager{clients: make(map[string]*redis.Client)}
}

// Client returns a verified, shared Redis client for an address.
// A failed readiness check is never cached.
func (m *RedisManager) Client(ctx context.Context, address string) (*redis.Client, error) {
	if m == nil {
		return nil, errors.New("Redis manager is nil")
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("Redis address is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errors.New("Redis manager is closed")
	}
	if c := m.clients[address]; c != nil {
		return c, nil
	}
	client := redis.NewClient(&redis.Options{Addr: address})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}
	m.clients[address] = client
	return client, nil
}

// Close releases every client once. It is safe to call repeatedly.
func (m *RedisManager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	clients := m.clients
	m.clients = nil
	m.mu.Unlock()

	var closeErrors []error
	for _, client := range clients {
		if err := client.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}
