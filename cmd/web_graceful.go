package cmd

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/graceful"
)

func runHTTP(
	network, listenAddr, name string,
	handler http.Handler,
) error {
	return graceful.HTTPListenAndServe(
		network,
		listenAddr,
		name,
		handler,
	)
}
