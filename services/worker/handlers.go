package worker

import (
	"fmt"

	"github.com/datvtph41107/bdspro/services/jobs"
)

type HandlerDependencies struct {
	VerifyEmailDelivery jobs.VerifyEmailDelivery

	ResetProofIssuer      jobs.ResetProofIssuer
	PasswordResetDelivery jobs.PasswordResetDelivery

	TeamInviteDelivery jobs.TeamInviteDelivery
}

func TaskHandlers(
	deps HandlerDependencies,
) ([]TaskHandler, error) {
	if deps.VerifyEmailDelivery == nil ||
		deps.ResetProofIssuer == nil ||
		deps.PasswordResetDelivery == nil ||
		deps.TeamInviteDelivery == nil {
		return nil, fmt.Errorf(
			"worker: business delivery dependencies incomplete",
		)
	}

	return []TaskHandler{
		{
			Type: jobs.TaskSendVerifyEmail,
			Handler: jobs.NewVerifyEmailHandler(
				deps.VerifyEmailDelivery,
			),
		},
		{
			Type: jobs.TaskSendPasswordReset,
			Handler: jobs.NewPasswordResetHandler(
				deps.ResetProofIssuer,
				deps.PasswordResetDelivery,
			),
		},
		{
			Type: jobs.TaskSendTeamInvite,
			Handler: jobs.NewTeamInviteHandler(
				deps.TeamInviteDelivery,
			),
		},
	}, nil
}
