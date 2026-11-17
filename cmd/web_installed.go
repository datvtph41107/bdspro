package cmd

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/process"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers"
)

var errWebStartupInterrupted = errors.New("web: startup interrupted by graceful shutdown")

// serveInstalled prepares and runs the installed Web application.
// Startup registration is deliberately retained in this first refactor:
// a later step will define an explicit startup-to-server ownership handoff.
func serveInstalled(ctx context.Context) error {
	startedAt := time.Now()
	log.Info("BDSPro version: %s%s", setting.AppVer, setting.AppBuiltWith)
	log.Info("HTTP runtime starting in %s mode", setting.Common.RunMode)

	g := graceful.GetManager()
	if err := g.RegisterServer(); err != nil {
		<-g.Done()
		if errors.Is(err, graceful.ErrShuttingDown) {
			log.Info("Web startup skipped: shutdown already requested")
			return nil
		}
		return err
	}

	runErr := func() error {
		defer g.ServerDone()

		webCtx, _, finished := process.GetManager().AddTypedContext(
			g.HammerContext(), "Web: HTTP API", process.SystemProcessType, true,
		)
		defer finished()

		isStopping := func() bool {
			return ctx.Err() != nil || g.ShutdownContext().Err() != nil
		}
		if isStopping() {
			return errWebStartupInterrupted
		}
		if err := routers.InitWebInstalled(webCtx); err != nil {
			return err
		}
		if isStopping() {
			return errWebStartupInterrupted
		}
		handler, err := routers.NormalHandler()
		if err != nil {
			return err
		}
		if isStopping() {
			return errWebStartupInterrupted
		}

		return graceful.HTTPListenAndServe(
			g, "tcp", setting.Server.HTTPServerAddress, "BDSPro Web",
			handler, setting.HTTPShutdownTimeout(),
			func(_ net.Addr) {
				log.Info("HTTP server ready in %s", time.Since(startedAt))
			},
		)
	}()

	g.DoGracefulShutdown()
	<-g.Done()

	if errors.Is(runErr, errWebStartupInterrupted) ||
		errors.Is(runErr, graceful.ErrShuttingDown) {
		return nil
	}
	if errors.Is(runErr, context.Canceled) && g.ShutdownContext().Err() != nil {
		return nil
	}
	return runErr
}
