package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datvtph41107/bdspro/api"
	db "github.com/datvtph41107/bdspro/db/sqlc"
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
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	store := db.NewStore(pool)
	server, err := api.NewServer(config, store)
	if err != nil {
		return err
	}
	return server.Start(ctx, config.ServerAddress)
}
