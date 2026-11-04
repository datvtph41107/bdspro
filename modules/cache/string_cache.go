// Package cache provides explicit cache semantics independently of its backend.
// A cache miss is different from a backend error; security state is not a cache.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/datvtph41107/bdspro/modules/json"
)

var (
	ErrInvalidTTL     = errors.New("cache TTL must be positive")
	ErrLoaderRequired = errors.New("cache loader is required")
)

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
	if c == nil {
		return "", errors.New("cache is required")
	}
	if load == nil {
		return "", ErrLoaderRequired
	}
	if ttl <= 0 {
		return "", ErrInvalidTTL
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

// PutJSON writes JSON with the same TTL and error semantics as string values.
func PutJSON(ctx context.Context, c StringCache, key string, value any, ttl time.Duration) error {
	if c == nil {
		return errors.New("cache is required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Put(ctx, key, string(data), ttl)
}

// GetJSON distinguishes a missing key from malformed JSON or backend outage.
func GetJSON(ctx context.Context, c StringCache, key string, target any) (bool, error) {
	if c == nil || target == nil {
		return false, errors.New("cache and JSON target are required")
	}
	value, hit, err := c.Get(ctx, key)
	if err != nil || !hit {
		return hit, err
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return false, err
	}
	return true, nil
}
