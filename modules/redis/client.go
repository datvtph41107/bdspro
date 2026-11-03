// Package redis owns Redis connectivity for request security and background jobs.
package redis

import (
    "context"
    "fmt"
    "time"

    redisclient "github.com/go-redis/redis/v8"
)

// Connect verifies readiness before returning the client to any runtime.
func Connect(ctx context.Context, address string) (*redisclient.Client, error) {
    if address == "" {return nil, fmt.Errorf("Redis address is required")}
    client := redisclient.NewClient(&redisclient.Options{Addr:address})
    pingCtx, cancel := context.WithTimeout(ctx,5*time.Second)
    defer cancel()
    if err := client.Ping(pingCtx).Err(); err != nil {
        _ = client.Close()
        return nil, fmt.Errorf("ping Redis: %w", err)
    }
    return client,nil
}
