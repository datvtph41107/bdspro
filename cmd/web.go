package cmd

import (
	"context"
	"errors"
	"net/http"

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

func runWeb(ctx context.Context, _ *cli.Command) error {
	managerCtx, cancel := context.WithCancel(ctx)
	graceful.InitManager(managerCtx)
	defer cancel()

	return serveInstalled()
}

// serveInstalled composes the Web application and
// waits for the process lifecycle to finish.
func serveInstalled() error {
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
			g.HammerContext(),
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

		return listen(handler)
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
func listen(handler http.Handler) error {
	g := graceful.GetManager()

	_, _, finished := process.GetManager().AddTypedContext(
		g.HammerContext(),
		"Web: BDSPro Server",
		process.SystemProcessType,
		true,
	)
	defer finished()

	address := setting.Server.HTTPServerAddress

	log.Info("Listen: http://%s", address)

	err := runHTTP(
		"tcp",
		address,
		"BDSPro Web",
		handler,
	)

	if err != nil &&
		!errors.Is(err, graceful.ErrShuttingDown) {
		log.Error("HTTP listener failed: %v", err)
	}

	log.Info("HTTP listener closed: %s", address)

	return err
}
