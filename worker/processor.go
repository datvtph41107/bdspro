package worker

import (
	"context"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type TaskProcessor interface {
	Start() error
	Shutdown()
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server         *asynq.Server
	store          db.Store
	mailer         mail.EmailSender
	verifyEmailURL string
}

func NewRedisTaskProcessor(options asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender, verifyEmailURL string) TaskProcessor {
	logger := NewLogger()
	redis.SetLogger(logger)
	server := asynq.NewServer(options, asynq.Config{
		Queues: map[string]int{
			QueueCritical: 10,
			QueueDefault:  5,
		},
		Logger:   logger,
		LogLevel: asynq.WarnLevel,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			taskID, _ := asynq.GetTaskID(ctx)
			retried, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)
			log.Error().Err(err).Str("task", task.Type()).Str("task_id", taskID).Int("retried", retried).Int("max_retry", maxRetry).Msg("task failed")
		}),
	})
	return &RedisTaskProcessor{
		server:         server,
		store:          store,
		mailer:         mailer,
		verifyEmailURL: verifyEmailURL,
	}
}

func (p *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskSendVerifyEmail, p.ProcessTaskSendVerifyEmail)
	return p.server.Start(mux)
}

func (p *RedisTaskProcessor) Shutdown() {
	p.server.Shutdown()
}
