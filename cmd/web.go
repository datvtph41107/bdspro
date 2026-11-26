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

func prepareWeb(command *cli.Command) (http.Handler, error) {
	if err := setting.LoadWebSettings(configPathOptions(command), os.Environ()); err != nil {
		return nil, err
	}
	if err := setting.InitLoggers(); err != nil {
		return nil, err
	}

	routes, err := routers.PrepareWebRoutes()
	if err != nil {
		return nil, fmt.Errorf("prepare HTTP routes: %w", err)
	}
	log.Info("Using configuration file: %s", setting.CustomConf)
	return routes, nil
}

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

	runErr := func() error {
		defer g.ServerDone()

		if g.ShutdownContext().Err() != nil {
			return nil
		}

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
