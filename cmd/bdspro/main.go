package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/datvtph41107/bdspro/bootstrap"
	"github.com/rs/zerolog/log"
)

// main owns only the process boundary: OS signals, cancellation and exit.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := bootstrap.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("BDSPro stopped")
	}
}
