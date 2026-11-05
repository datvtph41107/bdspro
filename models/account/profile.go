package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// ProfileQueries is the SQLC capability used by the signed-in account profile.
// A client-supplied person_id or account_id must never authorize self-service.
type ProfileQueries interface {
	GetAccountProfile(context.Context, int64) (db.GetAccountProfileRow, error)
	UpdatePersonProfile(context.Context, db.UpdatePersonProfileParams) (int64, error)
	UpdateAccountSettings(context.Context, db.UpdateAccountSettingsParams) (int64, error)
}

func GetProfile(ctx context.Context, q ProfileQueries, accountID int64) (db.GetAccountProfileRow, error) {
	if accountID <= 0 {
		return db.GetAccountProfileRow{}, ErrAccountUnavailable
	}
	p, err := q.GetAccountProfile(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetAccountProfileRow{}, ErrAccountUnavailable
	}
	return p, err
}

func UpdateProfile(ctx context.Context, q ProfileQueries, arg db.UpdatePersonProfileParams) error {
	if arg.AccountID <= 0 {
		return ErrAccountUnavailable
	}
	n, err := q.UpdatePersonProfile(ctx, arg)
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrAccountUnavailable
	}
	return nil
}

func UpdateSettings(ctx context.Context, q ProfileQueries, arg db.UpdateAccountSettingsParams) error {
	if arg.AccountID <= 0 {
		return ErrAccountUnavailable
	}
	n, err := q.UpdateAccountSettings(ctx, arg)
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrAccountUnavailable
	}
	return nil
}
