package main

import (
	"context"
	"fmt"
	"log"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

// main may terminate the process only after run has released its resources.
// log.Fatal inside run would call os.Exit and skip deferred pool.Close.
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

	pool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		return fmt.Errorf("cannot connect to db: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("cannot reach database: %w", err)
	}

	mailer := auth.NewSMTPMailer(
		config.SMTPAddr, config.SMTPUser, config.SMTPPassword,
		config.SMTPFrom, config.VerificationURL,
	)
	server := api.NewServer(
		pool, auth.New(pool, mailer),
		organization.New(pool), source.NewService(pool),
		config.Environment == "development", config.AllowedOrigin,
	)
	if config.AccessTokenKey != "" {
		issuer, err := auth.NewAccessTokens(config.AccessTokenKey)
		if err != nil {
			return fmt.Errorf("invalid access token configuration: %w", err)
		}
		server.EnableAccessTokens(issuer)
	}

	if err := server.Start(config.ServerAddress); err != nil {
		return fmt.Errorf("cannot start server: %w", err)
	}
	return nil
}
