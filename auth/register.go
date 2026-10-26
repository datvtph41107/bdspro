package auth

import (
    "context"
    "errors"
    "strings"

    apperrors "github.com/datvtph41107/bdspro/app/errors"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/util"
    "github.com/jackc/pgx/v5/pgconn"
)

func Register(ctx context.Context, store db.Store, scheduler VerificationTaskScheduler, email, password string) (db.Account, error) {
	hash, err := util.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	account, err := store.RegisterTx(ctx, db.RegisterTxParams{
		Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: hash,
		AfterCreate: func(account db.Account) error {
			return scheduler.ScheduleVerifyEmail(ctx, account.ID)
		},
	})
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
            return db.Account{}, apperrors.Wrap(apperrors.EmailRegistered, err)
        }
        return db.Account{}, err
    }
    return account, nil
}
