package cmd

import (
	"context"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/modules/graceful"
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
	log.Info("BDSPro version: %s%s", setting.AppVer, setting.AppBuiltWith)
	log.Info("HTTP runtime starting in %s mode", setting.Common.RunMode)

	graceful.InitManagerWithTimeout(ctx, setting.Common.Graceful.HammerTimeout)
	manager := graceful.GetManager()

	server, err := routers.InitWebInstalled(manager.HammerContext())
	if err != nil {
		manager.DoGracefulShutdown()
		<-manager.Done()
		return err
	}
	// Resource cleanup follows HTTP drain, not the initial cancellation.
	defer server.Close()

	manager.RegisterServer()
	serveErr := func() error {
		defer manager.ServerDone()
		return serveHTTP(manager.ShutdownContext(), server.Handler(), setting.Server.HTTPServerAddress, func(_ net.Addr) {
			log.Info("HTTP server ready in %s", time.Since(startedAt))
		})
	}()

	manager.DoGracefulShutdown()
	<-manager.Done()
	return serveErr
}
