package db

import (
    "context"
    "errors"

    "github.com/jackc/pgx/v5"
)

var ErrEmailAlreadyVerified = errors.New("email already verified")

// An account row lock serializes concurrent verification task handlers.
// All SQL originates from db/query/*.sql.
func (s *SQLStore) GetOrCreateVerifyEmailTx(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error) {
    var record VerifyEmail
    err := s.execTx(ctx, func(q *Queries) error {
        verified, err := q.LockVerifyEmailAccountQuery(ctx, arg.AccountID)
        if err != nil { return err }
        if verified { return ErrEmailAlreadyVerified }

        record, err = q.FindReusableVerifyEmailQuery(ctx, FindReusableVerifyEmailQueryParams{
            AccountID:arg.AccountID, Email:arg.Email,
        })
        if err == nil { return nil }
        if !errors.Is(err, pgx.ErrNoRows) { return err }
        record, err = q.CreateVerifyEmail(ctx, arg)
        return err
    })
    return record, err
}
