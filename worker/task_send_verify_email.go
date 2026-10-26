package worker

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

    "github.com/datvtph41107/bdspro/jobs"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
)

const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	AccountID int64 `json:"account_id"`
}

func (d *RedisTaskDistributor) DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal verification task: %w", err)
	}
	task := asynq.NewTask(TaskSendVerifyEmail, body, opts...)
	info, err := d.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Str("type", task.Type()).Str("queue", info.Queue).Str("task_id", info.ID).Int64("account_id", payload.AccountID).Msg("task enqueued")
	return nil
}

func (p *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    var payload PayloadSendVerifyEmail
    if err := json.Unmarshal(task.Payload(), &payload); err != nil {
        return fmt.Errorf("decode verification task: %w", errors.Join(err, asynq.SkipRetry))
    }
    if err := jobs.SendVerificationEmail(ctx, p.store, p.mailer, p.verifyEmailURL, payload.AccountID); err != nil {
        return err
    }
    taskID, _ := asynq.GetTaskID(ctx)
    log.Info().Str("type", task.Type()).Str("task_id", taskID).
        Int64("account_id", payload.AccountID).Msg("verification_email.completed")
    return nil
}
