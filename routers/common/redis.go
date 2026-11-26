package common

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
)

func InitRedis(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := conn.InitRedis(ctx); err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}

	if err := ctx.Err(); err != nil {
		conn.CloseRedis()
		return err
	}

	log.Info("Redis ready")
	return nil
}
