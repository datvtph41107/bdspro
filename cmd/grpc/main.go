// Command grpc starts the BDSPro RPC transport on its own local-only
// listener. Gin remains a separate, fully supported HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/gapi"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := util.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if config.GRPCServerAddress == "" {
		return errors.New("GRPC_SERVER_ADDRESS is required")
	}
	// util.Config rejects non-loopback gRPC addresses; this stage uses
	// insecure HTTP/2 and must not be accidentally reachable off-host.
	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}
	defer listener.Close()

	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return fmt.Errorf("JWT config: %w", err)
	}
	pool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = pool.Ping(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("database not ready: %w", err)
	}
	grpcServer, err := gapi.NewServer(maker, auth.New(pool, nil), organization.New(pool))
	if err != nil {
		return err
	}
	stop, cancelSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelSignals()
	done := make(chan error, 1)
	go func() { done <- grpcServer.Serve(listener) }()
	log.Printf("BDSPro gRPC listening on %s", listener.Addr())
	select {
	case err = <-done:
		return err
	case <-stop.Done():
		// GracefulStop can wait for in-flight requests. Bound shutdown and
		// force Stop when the grace period elapses.
		timeout := time.AfterFunc(5*time.Second, grpcServer.Stop)
		defer timeout.Stop()
		grpcServer.GracefulStop()
		err = <-done
		return err
	}
}
