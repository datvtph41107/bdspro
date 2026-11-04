// Package cache provides explicit cache semantics independently of its backend.
// A cache miss is different from a backend error; security state is not a cache.
package cache

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidTTL = errors.New("cache TTL must be positive")

// StringCache stores bounded-lifetime values. Get reports an actual miss as
// (empty, false, nil); it never hides a backend outage as a cache miss.
type StringCache interface {
	Get(ctx context.Context, key string) (value string, hit bool, err error)
	Put(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// GetString is explicit cache-aside loading. Backend errors are returned to
// the caller; each consumer owns its degradation/fallback policy.
func GetString(ctx context.Context, c StringCache, key string, ttl time.Duration, load func(context.Context) (string, error)) (string, error) {
	if c == nil || load == nil {
		return "", errors.New("cache and loader are required")
	}
	value, hit, err := c.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if hit {
		return value, nil
	}
	value, err = load(ctx)
	if err != nil {
		return "", err
	}
	if err := c.Put(ctx, key, value, ttl); err != nil {
		return "", err
	}
	return value, nil
}
