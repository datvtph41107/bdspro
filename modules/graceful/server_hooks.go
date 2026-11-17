package graceful

import (
	"context"
	"fmt"
	"time"
)

// awaitShutdown subscribes to Manager phase signals, rather than being called
// directly by Manager.doShutdown. This is the Server/Manager ownership boundary.
func (srv *Server) awaitShutdown() {
	defer close(srv.awaitDone)
	select {
	case <-srv.manager.IsShutdown():
	case <-srv.manager.IsHammer():
	}

	// Shutdown may block while HTTP waits for active connections. Observe
	// Hammer concurrently so escalation is not blocked by Shutdown itself.
	go func() {
		srv.doShutdown()
		close(srv.shutdownDone)
	}()

	select {
	case <-srv.manager.IsHammer():
		srv.doHammer()
	case <-srv.serveFinished:
	}
}

// doShutdown stops admitting connections, then performs protocol-specific
// graceful drain. On timeout, force-close transport connections.
func (srv *Server) doShutdown() {
	srv.lock.Lock()
	if srv.state != stateRunning {
		srv.lock.Unlock()
		return
	}
	srv.state = stateShuttingDown
	srv.lock.Unlock()

	// Unlike Gitea's listener-only doShutdown, BDSPro also delegates to
	// net/http's Shutdown for its established HTTP connection semantics.
	_ = srv.listener.Close()
	if srv.OnShutdown == nil {
		return
	}
	timeout := srv.ShutdownTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.OnShutdown(ctx); err != nil {
		srv.addShutdownError(fmt.Errorf("shutdown %s: %w", srv.name, err))
		srv.doHammer()
	}
}

// doHammer closes transport connections but cannot kill a stuck Go handler.
// The activity registration remains held until OnDrained returns.
func (srv *Server) doHammer() {
	srv.hammerOnce.Do(func() {
		if srv.OnHammer != nil {
			if err := srv.OnHammer(); err != nil {
				srv.addShutdownError(fmt.Errorf("hammer %s: %w", srv.name, err))
			}
		}
	})
}
