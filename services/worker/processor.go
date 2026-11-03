package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

// RedisTaskProcessor owns queue lifecycle and handler registration.
// Jobs implement the native asynq.Handler contract.
type RedisTaskProcessor struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewRedisTaskProcessor(options asynq.RedisClientOpt) *RedisTaskProcessor {
	logger := NewLogger()
	redis.SetLogger(logger)
	server := asynq.NewServer(options, asynq.Config{
		Queues: map[string]int{
			QueueCritical: 10,
			QueueDefault:  5,
		},
		Logger:       logger,
		LogLevel:     asynq.WarnLevel,
		ErrorHandler: asynq.ErrorHandlerFunc(logTaskFailure),
	})
	return &RedisTaskProcessor{server: server, mux: asynq.NewServeMux()}
}

func (p *RedisTaskProcessor) Handle(taskType string, handler asynq.Handler) {
	p.mux.Handle(taskType, handler)
}

func (p *RedisTaskProcessor) Start() error {
	return p.server.Start(p.mux)
}

func (p *RedisTaskProcessor) Shutdown() {
	p.server.Shutdown()
}

// logTaskFailure is the single diagnostic event for a failed task attempt.
// The error text and task payload may contain private account data.
func logTaskFailure(ctx context.Context, task *asynq.Task, err error) {
	taskID, _ := asynq.GetTaskID(ctx)
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	log.Ctx(ctx).Error().
		Str("task", task.Type()).
		Str("task_id", taskID).
		Str("error_type", fmt.Sprintf("%T", err)).
		Bool("skip_retry", errors.Is(err, asynq.SkipRetry)).
		Int("retried", retried).
		Int("max_retry", maxRetry).
		Msg("task.failed")
}
