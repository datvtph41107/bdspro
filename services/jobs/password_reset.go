package jobs

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/hibiken/asynq"
)

// ResetProof contains a newly generated plaintext
// proof and its one-way hash.
//
// Only the hash is persisted in PostgreSQL.
type ResetProof struct {
	Plaintext string
	Hash      []byte
}

type ResetProofIssuer func() (ResetProof, error)

type PasswordResetDelivery func(
	context.Context,
	int64,
	string,
) error

func NewPasswordResetHandler(
	issueProof ResetProofIssuer,
	deliver PasswordResetDelivery,
) asynq.Handler {
	if issueProof == nil || deliver == nil {
		panic("jobs: incomplete password-reset dependencies")
	}

	return asynq.HandlerFunc(
		func(ctx context.Context, task *asynq.Task) error {
			var payload PasswordResetPayload

			if err := decodeTask(task, &payload); err != nil {
				return err
			}

			if payload.AccountID <= 0 ||
				payload.ResetVersion < 0 {
				return fmt.Errorf(
					"password-reset: invalid payload",
				)
			}

			// The caller-supplied implementation must
			// use a cryptographically secure generator.
			proof, err := issueProof()
			if err != nil {
				return err
			}

			if proof.Plaintext == "" ||
				len(proof.Hash) == 0 {
				return fmt.Errorf(
					"password-reset: invalid generated proof",
				)
			}

			var issued bool

			err = db.WithTx(
				ctx,
				func(txCtx context.Context) error {
					queries, err := db.GetQueries(txCtx)
					if err != nil {
						return err
					}

					count, err :=
						queries.IssuePasswordResetTokenIfCurrentVersion(
							txCtx,
							db.IssuePasswordResetTokenIfCurrentVersionParams{
								AccountID:    payload.AccountID,
								ResetVersion: payload.ResetVersion,
								TokenHash:    proof.Hash,
							},
						)
					if err != nil {
						return err
					}

					issued = count == 1
					return nil
				},
			)
			if err != nil {
				return err
			}

			if !issued {
				// Stale reset intent.
				return nil
			}

			return deliver(
				ctx,
				payload.AccountID,
				proof.Plaintext,
			)
		},
	)
}
