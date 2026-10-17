package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	dbsqlc "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
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
		return fmt.Errorf("cannot load config: %w", err)
	}
	if config.DBDriver != "postgres" {
		return fmt.Errorf("unsupported DB_DRIVER %q", config.DBDriver)
	}
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE required")
	}
	if config.ServerAddress == "" {
		return fmt.Errorf("SERVER_ADDRESS required")
	}
	host, _, err := net.SplitHostPort(config.ServerAddress)
	if err != nil {
		return fmt.Errorf("invalid SERVER_ADDRESS: %w", err)
	}
	dev := config.Environment == "development"
	if dev && host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return fmt.Errorf("development verification tokens require a loopback listener")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()
	ping, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(ping)
	cancel()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	var mailer auth.Mailer
	if config.SMTPAddr != "" {
		mailer = auth.SMTPMailer{
			Addr: config.SMTPAddr, User: config.SMTPUser,
			Password: config.SMTPPassword, From: config.SMTPFrom,
			VerificationURL: config.VerificationURL,
		}
	}
	if !dev && mailer == nil {
		return fmt.Errorf("SMTP_ADDR required outside development")
	}
	if !dev && config.AllowedOrigin == "" {
		return fmt.Errorf("ALLOWED_ORIGIN required outside development")
	}
	authentication := auth.New(pool, mailer)
	app := api.NewServer(
		dbsqlc.NewStore(pool), authentication,
		organization.New(pool), source.NewService(pool),
		dev, config.AllowedOrigin,
	)
	server := &http.Server{
		Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	ln, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		return err
	}
	log.Printf("BDSPro listening on %s", ln.Addr())
	signals, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ln) }()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-signals.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return err
		}
	}
	return nil
}
