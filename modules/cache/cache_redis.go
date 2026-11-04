package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
)

// redisCache implements cache semantics over a borrowed Redis client. The
// connection manager (not this adapter) owns the client shutdown lifecycle.
type redisCache struct {
	client redis.Cmdable
	prefix string
}

func NewRedisStringCache(client redis.Cmdable, prefix string) (StringCache, error) {
	if client == nil {
		return nil, errors.New("Redis cache client is required")
	}
	if prefix == "" {
		return nil, errors.New("Redis cache prefix is required")
	}
	return &redisCache{client: client, prefix: prefix}, nil
}

func (c *redisCache) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := c.client.Get(ctx, c.prefix+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read Redis cache: %w", err)
	}
	return value, true, nil
}

func (c *redisCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if err := c.client.Set(ctx, c.prefix+key, value, ttl).Err(); err != nil {
		return fmt.Errorf("write Redis cache: %w", err)
	}
	return nil
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, c.prefix+key).Err(); err != nil {
		return fmt.Errorf("delete Redis cache: %w", err)
	}
	return nil
}
