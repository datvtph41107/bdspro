package auth

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/datvtph41107/bdspro/app/errors"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidCredentials = apperrors.InvalidCredentials

func Login(ctx context.Context, store db.Store, maker token.Maker, config util.Config, email, password, userAgent, clientIP string) (LoginResult, error) {
	var result LoginResult
	account, err := store.GetAccountByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	if err := util.CheckPassword(password, account.PasswordHash); err != nil {
		return result, ErrInvalidCredentials
	}

	return IssueSession(ctx, store, maker, config, account, userAgent, clientIP)
}
