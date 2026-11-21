package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

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

// runWeb owns the process lifecycle, not configuration parsing.
// prepareWeb publishes validated settings and returns compiled routes before
// resource startup; there is no second runtime WebConfig authority.
func runWeb(ctx context.Context, command *cli.Command) error {
	routes, err := prepareWeb(command)
	if err != nil {
		return err
	}

	managerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	graceful.InitManager(managerCtx)

	return serveInstalled(routes)
}

// prepareWeb performs command-specific initialization, not config parsing.
// Settings are published once; routers and process-owned resources read that
// same canonical setting package, as in Gitea's installed-app startup.
func prepareWeb(command *cli.Command) (http.Handler, error) {
	if err := setting.LoadWebSettings(configPathOptions(command), os.Environ()); err != nil {
		return nil, err
	}
	if err := setting.InitLoggers(); err != nil {
		return nil, err
	}
	// Registration errors still occur before PostgreSQL/Redis are opened.
	routes, err := routers.PrepareWebRoutes()
	if err != nil {
		return nil, fmt.Errorf("prepare HTTP routes: %w", err)
	}
	log.Info("Using configuration file: %s", setting.CustomConf)
	return routes, nil
}

// serveInstalled composes the Web application and
// waits for the process lifecycle to finish.
func serveInstalled(routes http.Handler) error {
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
		if err := routers.InitWebInstalled(g.ShutdownContext()); err != nil {
			if g.ShutdownContext().Err() != nil && errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		handler, err := routers.NormalHandler(routes)
		if err != nil {
			return err
		}

		if g.ShutdownContext().Err() != nil {
			return nil
		}

		return listen(handler)
	}()

	g.DoGracefulShutdown()
	<-g.Done()

	if errors.Is(runErr, graceful.ErrShuttingDown) {
		return nil
	}

	return runErr
}

// listen owns command-level listener tracking; modules/graceful owns HTTP
// shutdown and connection draining.
func listen(handler http.Handler) error {
	g := graceful.GetManager()
	address := setting.Server.HTTPServerAddress

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
		setting.HTTPShutdownTimeout(),
	)

	if err != nil &&
		!errors.Is(err, graceful.ErrShuttingDown) {
		log.Error("HTTP listener failed: %v", err)
	}

	log.Info("HTTP listener closed: %s", address)

	return err
}
