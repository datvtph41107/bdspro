package graceful

import (
	"errors"
	"log"
	"net"
)

// awaitShutdown observes lifecycle phases managed by Manager.
// Server owns the reaction for its own listener.
func (srv *Server) awaitShutdown() {
	select {
	case <-srv.manager.IsShutdown():
		srv.doShutdown()

	case <-srv.manager.IsHammer():
		srv.doShutdown()
		srv.doHammer()
	}

	// The Manager completes Hammer either when the
	// timeout expires or after registered activities
	// finish. This also releases the watcher.
	<-srv.manager.IsHammer()
	srv.doHammer()
}

// doShutdown executes the listener shutdown exactly once.
// Concurrent calls wait until the operation completes.
func (srv *Server) doShutdown() {
	srv.shutdownOnce.Do(func() {
		srv.lock.Lock()

		if srv.state != stateRunning {
			srv.lock.Unlock()
			return
		}

		srv.state = stateShuttingDown
		listener := srv.listener

		srv.lock.Unlock()

		// Protocol-specific shutdown preparation.
		if srv.OnShutdown != nil {
			srv.OnShutdown()
		}

		// Closing the listener stops accepting new
		// connections but leaves accepted connections
		// under Server connection tracking.
		if listener != nil {
			if err := listener.Close(); err != nil &&
				!errors.Is(err, net.ErrClosed) {
				log.Printf(
					"graceful: close listener: %v",
					err,
				)
			}
		}
	})
}

// doHammer releases the connection-drain condition.
// It does not guarantee handler goroutines exited.
func (srv *Server) doHammer() {
	if srv.getState() != stateShuttingDown {
		return
	}

	srv.closeAllConnections()
}
