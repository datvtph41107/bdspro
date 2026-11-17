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
	"github.com/urfave/cli/v3"
)

var errWebStartupInterrupted = errors.New("web: startup interrupted by graceful shutdown")

func newWebCommand() *cli.Command {
	return &cli.Command{
		Name:    "web",
		Aliases: []string{"serve"},
		Usage:   "Start the BDSPro HTTP API",
		Before:  rejectArguments("web"),
		Action:  runWeb,
	}
}

// runWeb is the application composition root; HTTP lifecycle implementation
// and resource cleanup stay with the owning packages.
func runWeb(ctx context.Context, _ *cli.Command) error {
	startedAt := time.Now()
	log.Info("BDSPro version: %s%s", setting.AppVer, setting.AppBuiltWith)
	log.Info("HTTP runtime starting in %s mode", setting.Common.RunMode)

	// Only the Web entrypoint initializes the process-wide Manager.
	graceful.InitManagerWithTimeout(ctx, setting.Common.Graceful.HammerTimeout)
	g := graceful.GetManager()

	// Startup may create DB/Redis AFTER SIGTERM. Hold a separate registered
	// activity until initialization and HTTP lifecycle both finish.
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

		// The Server registers its own HTTP-serving activity and pairs its
		// ServerDone only after active handlers have drained. This startup
		// activity remains registered until the Server returns.
		return graceful.HTTPListenAndServe(
			g, "tcp", setting.Server.HTTPServerAddress, "BDSPro Web",
			handler, setting.HTTPShutdownTimeout(),
			func(_ net.Addr) {
				log.Info("HTTP server ready in %s", time.Since(startedAt))
			},
		)
	}()

	// Error exits and normal stop converge on one idempotent lifecycle.
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
