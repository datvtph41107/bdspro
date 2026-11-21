package common

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
)

// InitRedis is the Web startup policy for the process-owned Redis client.
// modules/conn owns the connection and registers its graceful cleanup.
// A canceled initialization must not publish a usable Redis dependency.
func InitRedis(ctx context.Context, address string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := conn.InitRedis(ctx, address); err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}

	if err := ctx.Err(); err != nil {
		conn.CloseRedis()
		return err
	}

	log.Info("Redis ready")
	return nil
}
