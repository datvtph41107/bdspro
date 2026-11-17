package graceful

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ServeFunction owns an accept loop (for HTTP, http.Server.Serve).
type ServeFunction = func(net.Listener) error

// Server owns the listener and drain for one long-lived server. Manager owns
// process-wide lifecycle; Server translates it into transport operations.
type Server struct {
	manager *Manager
	network string
	address string
	name    string

	ShutdownTimeout time.Duration
	OnListening     func(net.Addr)
	OnShutdown      func(context.Context) error
	OnHammer        func() error
	OnDrained       func()

	lock        sync.Mutex
	state       state
	listener    net.Listener
	shutdownErr error
	hammerOnce  sync.Once

	shutdownDone  chan struct{}
	serveFinished chan struct{}
	awaitDone     chan struct{}
}

// NewServer does not open sockets or register with Manager; those are done in
// ListenAndServe/Serve. Explicit manager injection avoids a hidden GetManager.
func NewServer(manager *Manager, network, address, name string) *Server {
	if manager == nil {
		panic("graceful: nil server manager")
	}
	return &Server{
		manager: manager, network: network, address: address, name: name,
		ShutdownTimeout: 10 * time.Second,
		shutdownDone:    make(chan struct{}),
		serveFinished:   make(chan struct{}),
		awaitDone:       make(chan struct{}),
	}
}

// ListenAndServe binds a port and delegates lifecycle ownership to Serve.
func (srv *Server) ListenAndServe(serve ServeFunction) error {
	if srv.manager.ShutdownContext().Err() != nil {
		return ErrShuttingDown
	}
	listener, err := net.Listen(srv.network, srv.address)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	srv.listener = listener
	return srv.Serve(serve)
}

// Serve mirrors Gitea's Server.Serve ownership boundary. ServerDone is issued
// only after shutdown processing AND admitted handlers have drained.
func (srv *Server) Serve(serve ServeFunction) error {
	if srv.listener == nil || serve == nil {
		return errors.New("graceful: Server.Serve requires listener and serve callback")
	}
	defer srv.listener.Close()
	if err := srv.manager.RegisterServer(); err != nil {
		return err
	}
	defer srv.manager.ServerDone()

	if srv.manager.ShutdownContext().Err() != nil {
		return ErrShuttingDown
	}
	srv.setState(stateRunning)
	go srv.awaitShutdown()

	if srv.OnListening != nil && srv.manager.ShutdownContext().Err() == nil {
		srv.OnListening(srv.listener.Addr())
	}

	serveErr := serve(srv.listener)
	// Unexpected accept-loop exits are also process shutdown requests.
	srv.manager.DoGracefulShutdown()

	<-srv.shutdownDone
	if srv.OnDrained != nil {
		srv.OnDrained()
	}
	close(srv.serveFinished)
	<-srv.awaitDone
	srv.setState(stateTerminate)

	srv.lock.Lock()
	shutdownErr := srv.shutdownErr
	srv.lock.Unlock()
	if errors.Is(serveErr, http.ErrServerClosed) || errors.Is(serveErr, net.ErrClosed) {
		serveErr = nil
	}
	if serveErr != nil {
		serveErr = fmt.Errorf("serve %s: %w", srv.name, serveErr)
	}
	return errors.Join(serveErr, shutdownErr)
}

func (srv *Server) setState(next state) {
	srv.lock.Lock()
	srv.state = next
	srv.lock.Unlock()
}

func (srv *Server) addShutdownError(err error) {
	if err == nil {
		return
	}
	srv.lock.Lock()
	srv.shutdownErr = errors.Join(srv.shutdownErr, err)
	srv.lock.Unlock()
}
