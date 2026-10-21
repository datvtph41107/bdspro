package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	QueueCritical       = "critical"
	TaskSendVerifyEmail = "task:send_verify_email"
)

type PayloadSendVerifyEmail struct {
	AccountID int64 `json:"account_id"`
}

type TaskDistributor interface {
	DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error
}

type RedisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(options asynq.RedisClientOpt) *RedisTaskDistributor {
	return &RedisTaskDistributor{client: asynq.NewClient(options)}
}

func (d *RedisTaskDistributor) DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal verification task: %w", err)
	}
	task := asynq.NewTask(TaskSendVerifyEmail, body, opts...)
	_, err = d.client.EnqueueContext(ctx, task)
	return err
}

func (d *RedisTaskDistributor) Close() error {
	return d.client.Close()
}
