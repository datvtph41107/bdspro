package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

type TeamInviteDelivery func(
	ctx context.Context,
	inviteID uuid.UUID,
	invite sqlc.GetTeamInviteForDeliveryRow,
) error

func NewTeamInviteHandler(
	deliver TeamInviteDelivery,
) asynq.Handler {
	if deliver == nil {
		panic("jobs: nil team-invite delivery")
	}

	return asynq.HandlerFunc(
		func(ctx context.Context, task *asynq.Task) error {
			var payload TeamInvitePayload

			if err := decodeTask(task, &payload); err != nil {
				return err
			}

			inviteID, err := uuid.Parse(payload.InviteID)
			if err != nil {
				return fmt.Errorf(
					"team-invite: invalid invite ID: %w",
					err,
				)
			}

			queries, err := db.GetQueries(ctx)
			if err != nil {
				return err
			}

			invite, err := queries.GetTeamInviteForDelivery(
				ctx,
				inviteID,
			)

			if errors.Is(err, pgx.ErrNoRows) {

				return nil
			}

			if err != nil {
				return err
			}

			return deliver(ctx, inviteID, invite)
		},
	)
}
