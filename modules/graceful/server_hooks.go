package graceful

import (
	"errors"
	"log"
	"net"
)

func (srv *Server) awaitShutdown() {
	select {
	case <-srv.manager.IsShutdown():
		srv.doShutdown()

	case <-srv.manager.IsHammer():
		srv.doShutdown()
		srv.doHammer()
	}

	<-srv.manager.IsHammer()
	srv.doHammer()
}

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

		if srv.OnShutdown != nil {
			srv.OnShutdown()
		}

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

func (srv *Server) doHammer() {
	if srv.getState() != stateShuttingDown {
		return
	}

	srv.closeAllConnections()
}
