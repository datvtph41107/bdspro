package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// runWeb owns the process lifecycle, not individual configuration mechanisms.
// Prepare returns a validated snapshot and compiled routes before resources open.
func runWeb(ctx context.Context, command *cli.Command) error {
	cfg, routes, err := prepareWeb(command)
	if err != nil {
		return err
	}

	managerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	graceful.InitManager(managerCtx, cfg.Common.Graceful.HammerTimeout)

	return serveInstalled(cfg, routes)
}

// prepareWeb is the Web command's composition boundary. It has one caller and
// intentionally stays beside runWeb, rather than becoming a generic cmd helper.
func prepareWeb(command *cli.Command) (setting.WebConfig, http.Handler, error) {
	cfg, err := setting.LoadWebConfig(configPathOptions(command), os.Environ())
	if err != nil {
		return setting.WebConfig{}, nil, err
	}
	if err := setting.InitLoggers(cfg.Common.Log); err != nil {
		return setting.WebConfig{}, nil, err
	}
	// Register routes before opening PostgreSQL or Redis. A route programmer
	// error must not create external resources or publish legacy HTTP globals.
	routes, err := routers.PrepareWebRoutes(cfg)
	if err != nil {
		return setting.WebConfig{}, nil, fmt.Errorf("prepare HTTP routes: %w", err)
	}
	setting.PublishLegacyWeb(cfg) // temporary bridge for existing HTTP consumers
	log.Info("Using configuration file: %s", cfg.Paths.ConfigFile)
	return cfg, routes, nil
}

// serveInstalled composes the Web application and
// waits for the process lifecycle to finish.
func serveInstalled(cfg setting.WebConfig, routes http.Handler) error {
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

		// Startup is not serving requests: cancel immediately on shutdown,
		// rather than waiting for the hammer phase used for HTTP draining.
		if err := routers.InitWebInstalled(
			g.ShutdownContext(), cfg,
		); err != nil {
			if g.ShutdownContext().Err() != nil && errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		handler, err := routers.NormalHandler(routes, cfg.IPActive.Policies)
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

	err := graceful.HTTPListenAndServe(
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
