package main

import (
	"context"
	"fmt"
	"log"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/datvtph41107/bdspro/token"
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
	authService := auth.New(pool, mailer)
	if config.PhoneOTPKey != "" {
		var sender auth.SMSSender
		if config.DevSMSOutbox {
			// Development-only transport; config validation fails closed in production.
			sender, err = auth.NewDevelopmentSMSOutbox(context.Background(), pool)
			if err != nil {
				return fmt.Errorf("cannot initialize development SMS outbox: %w", err)
			}
		} else {
			sender = auth.NewTwilioSMS(config.TwilioAccountSID, config.TwilioAuthToken, config.TwilioFrom)
		}
		if err := authService.ConfigurePhoneVerification(config.PhoneOTPKey, sender); err != nil {
			return fmt.Errorf("invalid phone OTP configuration: %w", err)
		}
	}
	server := api.NewServer(
		pool, authService,
		organization.New(pool), source.NewService(pool),
		config.Environment == "development", config.AllowedOrigin,
	)
	if config.AccessTokenKey != "" {
		maker, err := token.NewJWTMaker(config.AccessTokenKey)
		if err != nil {
			return fmt.Errorf("invalid JWT configuration: %w", err)
		}
		server.SetJWTMaker(maker)
	}

	if err := server.Start(config.ServerAddress); err != nil {
		return fmt.Errorf("cannot start server: %w", err)
	}
	return nil
}
