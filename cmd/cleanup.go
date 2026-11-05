package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/services/worker"
	"github.com/urfave/cli/v3"
)

// cleanup is a one-shot PostgreSQL maintenance command. Unlike the long-running
// queue worker, it has no Redis or Asynq startup dependency.
func newCleanupCommand() *cli.Command {
	return &cli.Command{
		Name:   "cleanup",
		Usage:  "Expire login history and audit events using PostgreSQL only",
		Before: PrepareConsoleLoggerLevel(log.INFO),
		Action: runCleanup,
	}
}

func runCleanup(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return fmt.Errorf("unexpected cleanup argument %q", command.Args().First())
	}
	cfg, err := loadSettings(command)
	if err != nil {
		return err
	}
	cleanupCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	store, err := db.InitStore(cleanupCtx, cfg.DBSource)
	if err != nil {
		return fmt.Errorf("open cleanup database: %w", err)
	}
	defer store.Close()

	deleted, err := worker.NewLoginRetention(store, cfg.AuditRetentionDays).Sweep(cleanupCtx, time.Now())
	if err != nil {
		return fmt.Errorf("cleanup login/audit history after %d deletions: %w", deleted, err)
	}
	log.Info("Login and audit cleanup completed, deleted=%d", deleted)
	return nil
}
