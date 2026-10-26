package bootstrap

import (
    "context"
    "fmt"
    "net"
    "time"

    httpserver "github.com/datvtph41107/bdspro/app/http/server"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/mail"
    "github.com/datvtph41107/bdspro/util"
    "github.com/datvtph41107/bdspro/worker"
    "github.com/go-redis/redis/v8"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
    "golang.org/x/sync/errgroup"
)

func runTaskProcessor(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	redisOpt asynq.RedisClientOpt,
	store db.Store,
) error {
	mailer := mail.NewSMTPSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword, config.SMTPAddress, config.SMTPAuth)
	taskProcessor := worker.NewRedisTaskProcessor(
		redisOpt,
		store,
		mailer,
		config.VerifyEmailURL,
	)
	if err := taskProcessor.Start(); err != nil {
		return fmt.Errorf("cannot start task processor: %w", err)
	}
	log.Info().Msg("task.processor.started")
	waitGroup.Go(func() error {
		<-ctx.Done()
		taskProcessor.Shutdown()
		log.Info().Msg("task.processor.stopped")
		return nil
	})
	return nil
}


func runGinServer(ctx context.Context, config util.Config, store db.Store, taskDistributor worker.TaskDistributor, redisClient *redis.Client, phoneOTP db.PhoneOTPRepository, startedAt time.Time) error {
	server, err := httpserver.NewServer(config, store, taskDistributor)
	if err != nil {
		return err
	}
	if err := server.EnableAuthRateLimiting(redisClient); err != nil { return err }
	if err := server.EnableGoogleOAuth(redisClient); err != nil { return fmt.Errorf("enable Google OAuth: %w", err) }
	if err := server.EnablePhoneOTP(phoneOTP); err != nil { return fmt.Errorf("enable phone OTP: %w", err) }
	return server.StartWithReady(ctx, config.HTTPServerAddress, func(net.Addr) {
		log.Info().Dur("duration", time.Since(startedAt)).Msg("startup.ready")
	})
}
