package graceful

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type ServeFunction = func(net.Listener) error

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

	err := serve(srv.listener)

	wasStopping :=
		srv.manager.ShutdownContext().Err() != nil

	srv.manager.DoGracefulShutdown()

	srv.doShutdown()

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

func (srv *Server) waitForActiveConnections() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	for srv.connCounter > 0 && !srv.hammered {
		srv.connEmptyCond.Wait()
	}
}

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

func (srv *Server) removeConnection() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	srv.connCounter--

	if srv.connCounter == 0 {
		srv.connEmptyCond.Broadcast()
	}
}

func (srv *Server) closeAllConnections() {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	srv.hammered = true
	srv.connEmptyCond.Broadcast()
}

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
