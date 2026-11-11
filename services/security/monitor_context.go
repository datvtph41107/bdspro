package security

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type monitorRedisKey struct{}

// Monitoring may degrade without changing authentication admission policy.
func WithMonitorRedis(ctx context.Context, client *redis.Client) context.Context {
	if ctx == nil {
		panic("security: nil request context")
	}
	return context.WithValue(ctx, monitorRedisKey{}, client)
}
func MonitorRedisFromContext(ctx context.Context) *redis.Client {
	if ctx == nil {
		return nil
	}
	client, _ := ctx.Value(monitorRedisKey{}).(*redis.Client)
	return client
}
