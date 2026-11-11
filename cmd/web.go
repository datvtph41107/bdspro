package cmd

import (
	"context"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/process"
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
	// Register before subsystem startup; SIGTERM during initialization
	// must not race a late registration against the shutdown transition.
	manager.RegisterServer()
	webCtx, _, finished := process.GetManager().AddTypedContext(
		manager.HammerContext(), "Web: HTTP API", process.SystemProcessType, true,
	)
	defer finished()

	err := routers.InitWebInstalled(webCtx)
	if err != nil {
		manager.ServerDone()
		manager.DoGracefulShutdown()
		<-manager.Done()
		return err
	}
	// DB and Redis terminate hooks run after the HTTP activity is drained.
	handler, err := routers.NormalHandler()
	if err != nil {
		manager.ServerDone()
		manager.DoGracefulShutdown()
		<-manager.Done()
		return err
	}

	serveErr := func() error {
		defer manager.ServerDone()
		return serveHTTP(manager.ShutdownContext(), handler, setting.Server.HTTPServerAddress, func(_ net.Addr) {
			log.Info("HTTP server ready in %s", time.Since(startedAt))
		})
	}()

	manager.DoGracefulShutdown()
	<-manager.Done()
	return serveErr
}
