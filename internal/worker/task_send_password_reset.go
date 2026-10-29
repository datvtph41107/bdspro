package worker

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

    "github.com/datvtph41107/bdspro/internal/jobs"
    "github.com/datvtph41107/bdspro/internal/mail"
    "github.com/hibiken/asynq"
)

// TaskSendPasswordReset is a persistent queue protocol identifier.
const TaskSendPasswordReset = "task:send_password_reset"

func NewPasswordResetHandler(store jobs.PasswordResetStore, sender mail.EmailSender, verifyURL string) asynq.Handler {
    return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
        var payload struct { AccountID int64 `json:"account_id"` }
        if err := json.Unmarshal(task.Payload(), &payload); err != nil {
            return fmt.Errorf("decode password reset task: %w", errors.Join(err, asynq.SkipRetry))
        }
        if payload.AccountID <= 0 {
            return fmt.Errorf("invalid password reset account: %w", asynq.SkipRetry)
        }
        return jobs.SendPasswordReset(ctx, store, sender, verifyURL, payload.AccountID)
    })
}
