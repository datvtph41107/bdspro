package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/services/worker"
	"github.com/urfave/cli/v3"
)

func newCleanupCommand() *cli.Command {
	return &cli.Command{
		Name: "cleanup", Usage: "Remove expired login, audit and OAuth records",
		Before: rejectArguments("cleanup"), Action: runCleanup,
	}
}

func runCleanup(ctx context.Context, _ *cli.Command) error {
	setting.LoadDBSetting()
	if err := setting.LoadAuditRetentionDays(); err != nil {
		return err
	}

	cleanupCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	store, err := db.InitStore(cleanupCtx, setting.Database.DBSource)
	if err != nil {
		return err
	}
	defer store.Close()

	now := time.Now()
	deleted, err := worker.NewLoginRetention(store, setting.Audit.RetentionDays).Sweep(cleanupCtx, now)
	if err != nil {
		return fmt.Errorf("cleanup login/audit history after %d deletions: %w", deleted, err)
	}
	log.Info("Login and audit cleanup completed, deleted=%d", deleted)

	oauthCounts, err := store.CleanupExpiredOAuth(cleanupCtx, now)
	if err != nil {
		return err
	}
	log.Info("OAuth cleanup completed, auth_codes=%d refresh_tokens=%d access_tokens=%d",
		oauthCounts.AuthCodes, oauthCounts.RefreshTokens, oauthCounts.AccessTokens)
	return nil
}
