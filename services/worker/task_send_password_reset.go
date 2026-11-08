package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/json"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/services/jobs"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/hibiken/asynq"
)

// TaskSendPasswordReset is a persisted task type; do not rename while tasks
// are outstanding in the outbox or Redis queues.
const TaskSendPasswordReset = "task:send_password_reset"

type PayloadSendPasswordReset struct {
	AccountID    int64  `json:"account_id"`
	ResetVersion *int64 `json:"reset_version"`
}

// NewPasswordResetHandler adapts the business job to native Asynq.
func NewPasswordResetHandler(store jobs.PasswordResetStore, sender mail.EmailSender, verifyURL string) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		var payload PayloadSendPasswordReset
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("decode password reset task: %w", errors.Join(err, asynq.SkipRetry))
		}
		if payload.AccountID <= 0 || (payload.ResetVersion != nil && *payload.ResetVersion < 0) {
			return fmt.Errorf("invalid password reset payload: %w", asynq.SkipRetry)
		}
		if payload.ResetVersion == nil {
			// Pre-migration jobs cannot prove they precede no completed recovery.
			// Fail closed without retrying or exposing account information.
			log.Ctx(ctx).Warn().Str("task", task.Type()).Msg("password_reset.legacy_job_skipped")
			return nil
		}
		return jobs.SendPasswordResetEmailForVersion(ctx, store, sender, verifyURL, payload.AccountID, *payload.ResetVersion)
	})
}
