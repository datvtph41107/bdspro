package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datvtph41107/bdspro/api"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/gapi"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

//go:embed doc/swagger/bdspro.swagger.json
var openAPI []byte

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
	redis := asynq.RedisClientOpt{Addr: config.RedisAddress}
	distributor := worker.NewRedisTaskDistributor(redis)
	defer distributor.Close()

	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	processor := worker.NewRedisTaskProcessor(redis, store, mailer, config.VerifyEmailURL)
	if err := processor.Start(); err != nil {
		return err
	}
	defer processor.Shutdown()

	server, err := api.NewServer(config, store, distributor)
	if err != nil {
		return err
	}
	grpcServer, err := gapi.NewServer(config, store, distributor)
	if err != nil {
		return err
	}
	mux := runtime.NewServeMux()
	if err := pb.RegisterBDSProHandlerServer(ctx, mux, grpcServer); err != nil {
		return err
	}
	server.MountGateway(mux)
	server.MountOpenAPI(openAPI)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return server.Start(ctx, config.ServerAddress) })
	group.Go(func() error { return grpcServer.Start(ctx, config.GRPCServerAddress) })
	return group.Wait()
}
