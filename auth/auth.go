package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func Register(ctx context.Context, store db.Store, distributor worker.TaskDistributor, email, password string) (db.Account, error) {
	hash, err := util.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	return store.RegisterTx(ctx, db.RegisterTxParams{
		Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: hash,
		AfterCreate: func(account db.Account) error {
			return distributor.DistributeTaskSendVerifyEmail(ctx,
				&worker.PayloadSendVerifyEmail{AccountID: account.ID},
				asynq.MaxRetry(10), asynq.ProcessIn(10*time.Second), asynq.Queue(worker.QueueCritical))
		},
	})
}

type LoginResult struct {
	Account          db.Account
	Session          db.Session
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

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

	accessToken, accessPayload, err := maker.CreateToken(account.ID, config.AccessTokenDuration, token.TokenTypeAccess)
	if err != nil {
		return result, err
	}
	refreshToken, refreshPayload, err := maker.CreateToken(account.ID, config.RefreshTokenDuration, token.TokenTypeRefresh)
	if err != nil {
		return result, err
	}
	session, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID: refreshPayload.ID, AccountID: account.ID, RefreshToken: refreshToken,
		UserAgent: userAgent, ClientIp: clientIP, ExpiresAt: refreshPayload.ExpiredAt,
	})
	if err != nil {
		return result, err
	}
	return LoginResult{
		Account: account, Session: session,
		AccessToken: accessToken, AccessExpiresAt: accessPayload.ExpiredAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshPayload.ExpiredAt,
	}, nil
}
