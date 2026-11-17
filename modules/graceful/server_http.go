package graceful

import (
	"context"
	"net"
	"net/http"
	"time"
)

// newHTTPServer adapts an HTTP handler to a graceful Server.
//
// The generic Server owns listener and connection lifecycle.
// net/http owns HTTP protocol handling.
func newHTTPServer(
	network, address, name string,
	handler http.Handler,
) (*Server, ServeFunction) {
	manager := GetManager()

	server := NewServer(
		manager,
		network,
		address,
		name,
	)

	httpServer := http.Server{
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return manager.HammerContext()
		},
		ReadHeaderTimeout: 10 * time.Second,
	}

	server.OnShutdown = func() {
		httpServer.SetKeepAlivesEnabled(false)
	}

	return server, httpServer.Serve
}

// HTTPListenAndServe constructs the HTTP adapter and
// delegates listening and serving to graceful.Server.
func HTTPListenAndServe(
	network, address, name string,
	handler http.Handler,
) error {
	server, serve := newHTTPServer(
		network,
		address,
		name,
		handler,
	)

	return server.ListenAndServe(serve)
}
