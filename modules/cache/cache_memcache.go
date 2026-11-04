package cache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

const memcacheRelativeTTLMax = 30 * 24 * 60 * 60

// memcacheCache uses Memcached's real client and expiration semantics.
// Memcached operations do not support context cancellation mid-command;
// a bounded client timeout prevents indefinite blocking.
type memcacheCache struct {
	client *memcache.Client
	prefix string
}

func NewMemcacheStringCache(host, prefix string) (StringCache, error) {
	if host == "" || prefix == "" {
		return nil, errors.New("Memcached host and key prefix are required")
	}
	servers := &memcache.ServerList{}
	if err := servers.SetServers(strings.Split(host, ";")...); err != nil {
		return nil, fmt.Errorf("invalid Memcached servers: %w", err)
	}
	client := memcache.NewFromSelector(servers)
	client.Timeout = 2 * time.Second
	return &memcacheCache{client: client, prefix: prefix}, nil
}

func validMemcacheKey(key string) bool {
	if len(key) == 0 || len(key) > 250 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] == 127 {
			return false
		}
	}
	return true
}

func (c *memcacheCache) key(key string) (string, error) {
	combined := c.prefix + key
	if !validMemcacheKey(combined) {
		return "", errors.New("invalid Memcached cache key")
	}
	return combined, nil
}

func memcacheExpiration(ttl time.Duration) (int32, error) {
	if ttl <= 0 {
		return 0, ErrInvalidTTL
	}
	seconds := int64(ttl / time.Second)
	if ttl%time.Second != 0 {
		seconds++
	}
	if seconds > memcacheRelativeTTLMax {
		seconds += time.Now().Unix()
	}
	if seconds > math.MaxInt32 {
		return 0, errors.New("Memcached expiration exceeds supported range")
	}
	return int32(seconds), nil
}

func (c *memcacheCache) Get(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	full, err := c.key(key)
	if err != nil {
		return "", false, err
	}
	item, err := c.client.Get(full)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read Memcached cache: %w", err)
	}
	return string(item.Value), true, nil
}

func (c *memcacheCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	full, err := c.key(key)
	if err != nil {
		return err
	}
	expiration, err := memcacheExpiration(ttl)
	if err != nil {
		return err
	}
	if err := c.client.Set(&memcache.Item{Key: full, Value: []byte(value), Expiration: expiration}); err != nil {
		return fmt.Errorf("write Memcached cache: %w", err)
	}
	return nil
}

func (c *memcacheCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	full, err := c.key(key)
	if err != nil {
		return err
	}
	if err := c.client.Delete(full); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		return fmt.Errorf("delete Memcached cache: %w", err)
	}
	return nil
}
