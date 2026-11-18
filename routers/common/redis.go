package common

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// InitRedis is the Web startup policy for the process-owned Redis client.
// modules/conn owns the connection; Web composition registers its cleanup.
// A canceled initialization must not publish a usable Redis dependency.
func InitRedis(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := conn.InitRedis(ctx, setting.Redis.Address); err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}

	graceful.GetManager().RunAtTerminate(conn.CloseRedis)

	if err := ctx.Err(); err != nil {
		conn.CloseRedis()
		return err
	}

	log.Info("Redis ready")
	return nil
}
