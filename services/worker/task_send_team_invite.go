package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/json"
	"github.com/datvtph41107/bdspro/services/jobs"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const TaskSendTeamInvite = "task:send_team_invite"

type PayloadSendTeamInvite struct {
	InviteID uuid.UUID `json:"invite_id"`
}

// NewTeamInviteHandler delivers a durable outbox intent. The payload contains
// an opaque UUID only. The worker derives the token from its stable key.
func NewTeamInviteHandler(sender mail.EmailSender, signingKey string) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		var payload PayloadSendTeamInvite
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("invalid team invitation payload: %w", errors.Join(err, asynq.SkipRetry))
		}
		if payload.InviteID == uuid.Nil {
			return fmt.Errorf("missing invitation ID: %w", asynq.SkipRetry)
		}
		return jobs.SendTeamInviteEmail(ctx, sender, signingKey, payload.InviteID)
	})
}
