package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

type TaskDistributor interface {
	DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error
}

type RedisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(options asynq.RedisClientOpt) *RedisTaskDistributor {
	return &RedisTaskDistributor{client: asynq.NewClient(options)}
}

func (d *RedisTaskDistributor) Close() error {
	return d.client.Close()
}
