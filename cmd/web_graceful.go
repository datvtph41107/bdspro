package cmd

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/graceful"
)

func runHTTP(
	network, listenAddr, name string,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	return graceful.HTTPListenAndServe(
		network,
		listenAddr,
		name,
		handler,
		shutdownTimeout,
	)
}
