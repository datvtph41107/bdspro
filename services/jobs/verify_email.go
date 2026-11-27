package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/db/sqlc"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

type VerifyEmailDelivery func(
	ctx context.Context,
	email string,
	code string,
) error

func NewVerifyEmailHandler(
	deliver VerifyEmailDelivery,
) asynq.Handler {
	if deliver == nil {
		panic("jobs: nil verify-email delivery")
	}

	return asynq.HandlerFunc(
		func(ctx context.Context, task *asynq.Task) error {
			var payload VerifyEmailPayload

			if err := decodeTask(task, &payload); err != nil {
				return err
			}

			if payload.AccountID <= 0 {
				return fmt.Errorf(
					"verify-email: invalid account ID",
				)
			}

			queries, err := db.GetQueries(ctx)
			if err != nil {
				return err
			}

			account, err := queries.ReadAccountByID(
				ctx,
				payload.AccountID,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}

			challenge, err :=
				queries.FindReusableVerifyEmailQuery(
					ctx,
					sqlc.FindReusableVerifyEmailQueryParams{
						AccountID: payload.AccountID,
						Email:     account.Email,
					},
				)

			if errors.Is(err, pgx.ErrNoRows) {

				return nil
			}
			if err != nil {
				return err
			}

			return deliver(
				ctx,
				challenge.Email,
				challenge.SecretCode,
			)
		},
	)
}
