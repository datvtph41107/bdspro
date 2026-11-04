package cache

import (
	"context"
	"sync"
	"time"
)

// ContextCache is request-scoped memoization, not a distributed cache.
// Mutating a value during a request requires explicit invalidation.
type ContextCache struct {
	mu      sync.RWMutex
	entries map[contextEntryKey]any
	expires time.Time
}

type contextEntryKey struct {
	group string
	key   any
}
type contextCacheKey struct{}

const contextCacheLifetime = 5 * time.Minute

func WithCacheContext(ctx context.Context) context.Context {
	if GetContextCache(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, contextCacheKey{}, &ContextCache{
		entries: make(map[contextEntryKey]any),
		expires: time.Now().Add(contextCacheLifetime),
	})
}

func GetContextCache(ctx context.Context) *ContextCache {
	c, _ := ctx.Value(contextCacheKey{}).(*ContextCache)
	return c
}

// GetWithContextCache memoizes a successful loader result for at most one
// request. It deliberately does not cache errors or coalesce concurrent loads.
func GetWithContextCache[T any, K comparable](
	ctx context.Context, group string, key K,
	load func(context.Context, K) (T, error),
) (T, error) {
	var zero T
	if load == nil {
		return zero, ErrLoaderRequired
	}
	c := GetContextCache(ctx)
	if c == nil {
		return load(ctx, key)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	id := contextEntryKey{group: group, key: key}
	c.mu.RLock()
	valid := time.Now().Before(c.expires)
	cached, ok := c.entries[id]
	c.mu.RUnlock()
	if valid && ok {
		if typed, validType := cached.(T); validType {
			return typed, nil
		}
	}
	value, err := load(ctx, key)
	if err != nil {
		return zero, err
	}
	c.mu.Lock()
	if time.Now().Before(c.expires) && ctx.Err() == nil {
		c.entries[id] = value
	}
	c.mu.Unlock()
	return value, nil
}

// DeleteContextCache invalidates a previously memoized value after a write.
func DeleteContextCache[K comparable](ctx context.Context, group string, key K) {
	if c := GetContextCache(ctx); c != nil {
		c.mu.Lock()
		delete(c.entries, contextEntryKey{group: group, key: key})
		c.mu.Unlock()
	}
}
