package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/internal/jobs"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// TaskSendVerifyEmail is a persisted Asynq task type. Do not rename without
// migrating queued tasks.
const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	AccountID int64 `json:"account_id"`
}

func NewSendVerifyEmailTask(accountID int64) (*asynq.Task, error) {
	if accountID <= 0 {
		return nil, errors.New("invalid verification account")
	}
	body, err := json.Marshal(PayloadSendVerifyEmail{AccountID: accountID})
	if err != nil {
		return nil, fmt.Errorf("encode verification task: %w", err)
	}
	return asynq.NewTask(TaskSendVerifyEmail, body), nil
}

// NewVerifyEmailHandler adapts the email verification job to native Asynq.
func NewVerifyEmailHandler(store jobs.VerificationStore, sender mail.EmailSender, url string) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		var payload PayloadSendVerifyEmail
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("decode verification task: %w", errors.Join(err, asynq.SkipRetry))
		}
		if payload.AccountID <= 0 {
			return fmt.Errorf("invalid verification account: %w", asynq.SkipRetry)
		}
		if err := jobs.SendVerificationEmail(ctx, store, sender, url, payload.AccountID); err != nil {
			return err
		}
		taskID, _ := asynq.GetTaskID(ctx)
		log.Ctx(ctx).Info().Str("task", task.Type()).Str("task_id", taskID).
			Msg("verification_email.completed")
		return nil
	})
}
