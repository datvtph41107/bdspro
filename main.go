package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/datvtph41107/bdspro/api"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	_ "github.com/datvtph41107/bdspro/doc/statik"
	"github.com/datvtph41107/bdspro/gapi"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/datvtph41107/bdspro/pb"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakyll/statik/fs"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

var interruptSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("BDSPro stopped")
	}
}

func run() error {
	config, err := util.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}
	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		return fmt.Errorf("cannot connect to db: %w", err)
	}
	defer connPool.Close()
	if err := connPool.Ping(ctx); err != nil {
		return fmt.Errorf("cannot ping db: %w", err)
	}

	if err := runDBMigration(config.MigrationURL, config.DBSource); err != nil {
		return err
	}

	store := db.NewStore(connPool)
	redisOpt := asynq.RedisClientOpt{Addr: config.RedisAddress}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	defer taskDistributor.Close()

	waitGroup, ctx := errgroup.WithContext(ctx)

	if err := runTaskProcessor(ctx, waitGroup, config, redisOpt, store); err != nil {
		return err
	}
	if err := runGatewayServer(ctx, waitGroup, config, store, taskDistributor); err != nil {
		return err
	}
	if err := runGrpcServer(ctx, waitGroup, config, store, taskDistributor); err != nil {
		return err
	}
	return waitGroup.Wait()
}

func runDBMigration(migrationURL, dbSource string) error {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}
	defer migration.Close()

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("cannot apply database migrations: %w", err)
	}
	log.Info().Msg("database migrations applied")
	return nil
}

func runTaskProcessor(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	redisOpt asynq.RedisClientOpt,
	store db.Store,
) error {
	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer, config.VerifyEmailURL)
	if err := taskProcessor.Start(); err != nil {
		return fmt.Errorf("cannot start task processor: %w", err)
	}
	log.Info().Msg("task processor started")
	waitGroup.Go(func() error {
		<-ctx.Done()
		taskProcessor.Shutdown()
		log.Info().Msg("task processor stopped")
		return nil
	})
	return nil
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	taskDistributor worker.TaskDistributor,
) error {
	server, err := gapi.NewServer(config, store, taskDistributor)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(gapi.GrpcLogger))
	pb.RegisterBDSProServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		return fmt.Errorf("cannot create gRPC listener: %w", err)
	}

	waitGroup.Go(func() error {
		log.Info().Str("address", listener.Addr().String()).Msg("gRPC server started")
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("gRPC server failed: %w", err)
		}
		return nil
	})
	waitGroup.Go(func() error {
		<-ctx.Done()
		grpcServer.GracefulStop()
		log.Info().Msg("gRPC server stopped")
		return nil
	})
	return nil
}

func runGatewayServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	taskDistributor worker.TaskDistributor,
) error {
	server, err := gapi.NewServer(config, store, taskDistributor)
	if err != nil {
		return err
	}
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions:   protojson.MarshalOptions{UseProtoNames: true},
		UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
	})
	grpcMux := runtime.NewServeMux(jsonOption)
	if err := pb.RegisterBDSProHandlerServer(ctx, grpcMux, server); err != nil {
		return fmt.Errorf("cannot register gRPC Gateway: %w", err)
	}

	// BDSPro's existing Gin-only routes remain available until their business
	// equivalents are introduced in protobuf. New /v1 routes use the Gateway.
	ginServer, err := api.NewServer(config, store, taskDistributor)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/", grpcMux)

	statikFS, err := fs.New()
	if err != nil {
		return fmt.Errorf("cannot open Swagger files: %w", err)
	}
	swaggerFile, err := statikFS.Open("/bdspro.swagger.json")
	if err != nil {
		return fmt.Errorf("cannot open Swagger JSON: %w", err)
	}
	swaggerJSON, err := io.ReadAll(swaggerFile)
	_ = swaggerFile.Close()
	if err != nil {
		return fmt.Errorf("cannot read Swagger JSON: %w", err)
	}
	mux.HandleFunc("/docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(swaggerJSON)
	})
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(statikFS)))
	mux.Handle("/", ginServer.Handler())

	c := cors.New(cors.Options{
		AllowedOrigins: config.AllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	httpServer := &http.Server{
		Addr:              config.HTTPServerAddress,
		Handler:           c.Handler(gapi.HttpLogger(mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	waitGroup.Go(func() error {
		log.Info().Str("address", httpServer.Addr).Msg("HTTP Gateway started")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP Gateway failed: %w", err)
		}
		return nil
	})
	waitGroup.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("cannot shut down HTTP Gateway: %w", err)
		}
		log.Info().Msg("HTTP Gateway stopped")
		return nil
	})
	return nil
}

func runGinServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) error {
	server, err := api.NewServer(config, store, taskDistributor)
	if err != nil {
		return err
	}
	return server.Start(context.Background(), config.HTTPServerAddress)
}
