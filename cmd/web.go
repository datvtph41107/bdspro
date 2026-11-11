package cmd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers"
	"github.com/urfave/cli/v3"
)

func newWebCommand() *cli.Command {
	return &cli.Command{
		Name: "web", Aliases: []string{"serve"},
		Usage:  "Start the BDSPro HTTP API",
		Before: rejectArguments("web"),
		Action: runWeb,
	}
}

func runWeb(ctx context.Context, _ *cli.Command) error {
	startedAt := time.Now()
	if err := setting.LoadSettings(); err != nil {
		return fmt.Errorf("load web settings: %w", err)
	}
	log.Info("BDSPro version: %s%s", setting.AppVer, setting.AppBuiltWith)
	log.Info("HTTP runtime starting in %s mode", setting.Common.RunMode)

	// Runtime resources are created by routers.InitWebInstalled and its
	// subsystem initializers, not in cmd. Its Close must release them in
	// reverse order after serveHTTP has drained active requests.
	server, err := routers.InitWebInstalled(ctx)
	if err != nil {
		return fmt.Errorf("initialize web runtime: %w", err)
	}
	defer server.Close()

	return serveHTTP(ctx, server.Handler(), setting.Server.HTTPServerAddress, func(_ net.Addr) {
		log.Info("HTTP server ready in %s", time.Since(startedAt))
	})
}
