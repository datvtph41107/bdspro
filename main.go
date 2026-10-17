package main

import (
	"context"
	"log"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/organization"
	"github.com/datvtph41107/bdspro/source"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	pool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("cannot reach database:", err)
	}

	store := db.NewStore(pool)
	mailer := auth.NewSMTPMailer(
		config.SMTPAddr, config.SMTPUser, config.SMTPPassword,
		config.SMTPFrom, config.VerificationURL,
	)
	server := api.NewServer(
		store, auth.New(pool, mailer),
		organization.New(pool), source.NewService(pool),
		config.Environment == "development", config.AllowedOrigin,
	)

	if err := server.Start(config.ServerAddress); err != nil {
		log.Fatal("cannot start server:", err)
	}
}
