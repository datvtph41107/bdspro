package jobs

import (
	"fmt"

	appjson "github.com/datvtph41107/bdspro/modules/json"
	"github.com/hibiken/asynq"
)

const (
	TaskSendVerifyEmail   = "task:send_verify_email"
	TaskSendPasswordReset = "task:send_password_reset"
	TaskSendTeamInvite    = "task:send_team_invite"
)

type VerifyEmailPayload struct {
	AccountID int64 `json:"account_id"`
}

type PasswordResetPayload struct {
	AccountID    int64 `json:"account_id"`
	ResetVersion int64 `json:"reset_version"`
}

type TeamInvitePayload struct {
	InviteID string `json:"invite_id"`
}

func decodeTask(
	task *asynq.Task,
	dst any,
) error {
	if task == nil {
		return fmt.Errorf("jobs: nil task")
	}

	if err := appjson.Unmarshal(
		task.Payload(),
		dst,
	); err != nil {
		return fmt.Errorf(
			"decode task payload: %w", err,
		)
	}

	return nil
}
