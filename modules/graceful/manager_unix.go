//go:build !windows

package graceful

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// start owns OS-specific signal subscription, registered before returning.
// CLI can independently cancel the parent context for other commands.
func (g *Manager) start() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(signals)
		g.handleSignals(g.ctx, signals)
	}()
}

// handleSignals converts Unix signals and CLI cancellation into a one-time
// shutdown request. Gitea's SIGHUP/reopen-log/USR2 policies are not enabled.
func (g *Manager) handleSignals(ctx context.Context, signals <-chan os.Signal) {
	select {
	case sig := <-signals:
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			g.DoGracefulShutdown()
		}
	case <-ctx.Done():
		g.DoGracefulShutdown()
	case <-g.Done():
		return
	}
}
