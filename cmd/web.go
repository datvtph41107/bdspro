package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
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
		Name:    "web",
		Aliases: []string{"serve"},
		Usage:   "Start the BDSPro HTTP API",
		Before:  rejectArguments("web"),
		Action:  runWeb,
	}
}

func runWeb(ctx context.Context, command *cli.Command) error {
	// A single validated snapshot is handed off to all resource constructors.
	cfg, err := setting.LoadWebConfig(configPathOptions(command), os.Environ())
	if err != nil {
		return err
	}
	if err := configureCommandLogger(cfg.Common.Log); err != nil {
		return err
	}
	// Only HTTP consumers still use these legacy globals; publish before serving.
	setting.PublishLegacyWeb(cfg)
	log.Info("Using configuration file: %s", cfg.Paths.ConfigFile)

	managerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	graceful.InitManager(managerCtx, cfg.Common.Graceful.HammerTimeout)

	return serveInstalled(cfg)
}

// serveInstalled composes the Web application and
// waits for the process lifecycle to finish.
func serveInstalled(cfg setting.WebConfig) error {
	g := graceful.GetManager()

	log.Info(
		"Starting BDSPro version: %s%s",
		setting.AppVer,
		setting.AppBuiltWith,
	)

	if err := g.RegisterServer(); err != nil {
		<-g.Done()
		if errors.Is(err, graceful.ErrShuttingDown) {
			return nil
		}
		return err
	}

	// Keep startup registered throughout this scope.
	// The inner HTTP server owns a separate registration.
	runErr := func() error {
		defer g.ServerDone()

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		if err := routers.InitWebInstalled(
			g.HammerContext(), cfg,
		); err != nil {
			return err
		}

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		handler, err := routers.NormalHandler()
		if err != nil {
			return err
		}

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		return listen(handler, cfg.Server.HTTPServerAddress, cfg.Common.Graceful.HTTPShutdownTimeout)
	}()

	g.DoGracefulShutdown()
	<-g.Done()

	if errors.Is(runErr, graceful.ErrShuttingDown) {
		return nil
	}

	return runErr
}

// listen prepares process tracking and delegates
// the actual HTTP lifecycle to runHTTP.
func listen(handler http.Handler, address string, shutdownTimeout time.Duration) error {
	g := graceful.GetManager()

	_, _, finished := process.GetManager().AddTypedContext(
		g.HammerContext(),
		"Web: BDSPro Server",
		process.SystemProcessType,
		true,
	)
	defer finished()

	log.Info("Listen: http://%s", address)

	err := runHTTP(
		"tcp",
		address,
		"BDSPro Web",
		handler,
		shutdownTimeout,
	)

	if err != nil &&
		!errors.Is(err, graceful.ErrShuttingDown) {
		log.Error("HTTP listener failed: %v", err)
	}

	log.Info("HTTP listener closed: %s", address)

	return err
}
