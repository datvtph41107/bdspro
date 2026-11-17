package graceful

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ServeFunction represents a protocol accept loop.
type ServeFunction = func(net.Listener) error

// Server owns one listener and its accepted connections.
// Manager owns the process lifecycle.
type Server struct {
	manager  *Manager
	network  string
	address  string
	name     string
	listener net.Listener

	lock          sync.Mutex
	state         state
	connCounter   int64
	connEmptyCond *sync.Cond
	hammered      bool

	shutdownOnce sync.Once

	OnShutdown func()
}

func NewServer(
	manager *Manager,
	network, address, name string,
) *Server {
	if manager == nil {
		panic("graceful: nil manager")
	}

	srv := &Server{
		manager: manager,
		network: network,
		address: address,
		name:    name,
		state:   stateInit,
	}

	srv.connEmptyCond = sync.NewCond(&srv.lock)

	return srv
}

// ListenAndServe creates the transport listener and
// delegates its lifecycle to Serve.
func (srv *Server) ListenAndServe(
	serve ServeFunction,
) error {
	if serve == nil {
		return errors.New("graceful: nil serve function")
	}

	if srv.manager.ShutdownContext().Err() != nil {
		return ErrShuttingDown
	}

	listener, err := net.Listen(
		srv.network,
		srv.address,
	)
	if err != nil {
		return fmt.Errorf(
			"listen %s: %w",
			srv.name,
			err,
		)
	}

	srv.listener = newWrappedListener(listener, srv)

	return srv.Serve(serve)
}

func (srv *Server) Serve(serve ServeFunction) error {
	if srv.listener == nil || serve == nil {
		return errors.New("graceful: invalid server")
	}

	if err := srv.manager.RegisterServer(); err != nil {
		_ = srv.listener.Close()
		return err
	}

	defer srv.manager.ServerDone()
	defer srv.listener.Close()

	srv.setState(stateRunning)

	go srv.awaitShutdown()

	// The listener accept loop belongs to this server.
	err := serve(srv.listener)

	// Distinguish a requested shutdown from
	// an unexpected serving failure.
	wasStopping :=
		srv.manager.ShutdownContext().Err() != nil

	// Serving has stopped. Request process shutdown
	// before waiting for accepted connections.
	srv.manager.DoGracefulShutdown()

	// Wait until OnShutdown and listener.Close
	// have completed, even when awaitShutdown
	// started the same operation concurrently.
	srv.doShutdown()

	// Drain accepted connections, or stop waiting
	// when the Manager reaches Hammer.
	srv.waitForActiveConnections()

	srv.setState(stateTerminate)

	if err == nil {
		return nil
	}

	if wasStopping &&
		(errors.Is(err, net.ErrClosed) ||
			errors.Is(err, http.ErrServerClosed)) {
		return nil
	}

	return fmt.Errorf(
		"serve %s: %w",
		srv.name,
		err,
	)
}

func (srv *Server) getState() state {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	return srv.state
}

func (srv *Server) setState(next state) {
	srv.lock.Lock()
	srv.state = next
	srv.lock.Unlock()
}

// waitForActiveConnections waits for actual connection
// completion, unless the Manager requests Hammer.
func (srv *Server) waitForActiveConnections() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	for srv.connCounter > 0 && !srv.hammered {
		srv.connEmptyCond.Wait()
	}
}

// wrapConnection admits one connection while running.
func (srv *Server) wrapConnection(
	conn net.Conn,
) (net.Conn, error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if srv.state != stateRunning {
		_ = conn.Close()
		return nil, net.ErrClosed
	}

	srv.connCounter++

	return &wrappedConn{
		Conn:   conn,
		server: srv,
	}, nil
}

// removeConnection decrements the count exactly once
// for every successfully admitted connection.
func (srv *Server) removeConnection() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	srv.connCounter--

	if srv.connCounter == 0 {
		srv.connEmptyCond.Broadcast()
	}
}

// closeAllConnections releases the drain wait.
// It does not claim that open connections are closed.
func (srv *Server) closeAllConnections() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	srv.hammered = true
	srv.connEmptyCond.Broadcast()
}

// wrappedListener tracks connections accepted by net/http.
type wrappedListener struct {
	net.Listener
	server *Server
}

func newWrappedListener(
	listener net.Listener,
	server *Server,
) *wrappedListener {
	return &wrappedListener{
		Listener: listener,
		server:   server,
	}
}

func (wl *wrappedListener) Accept() (net.Conn, error) {
	var conn net.Conn

	// Follow Gitea's TCP keep-alive handling.
	if tcpListener, ok := wl.Listener.(*net.TCPListener); ok {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			return nil, err
		}

		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(
			3 * time.Minute,
		)

		conn = tcpConn
	} else {
		accepted, err := wl.Listener.Accept()
		if err != nil {
			return nil, err
		}

		conn = accepted
	}

	return wl.server.wrapConnection(conn)
}

// wrappedConn reports closing to its owner exactly once.
type wrappedConn struct {
	net.Conn
	server *Server
	once   sync.Once
}

func (conn *wrappedConn) Close() error {
	err := conn.Conn.Close()

	conn.once.Do(func() {
		conn.server.removeConnection()
	})

	return err
}
