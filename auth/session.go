package auth

import (
	"context"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
)

type LoginResult struct {
	Account          db.Account
	Session          db.Session
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// IssueSession gives any already-authenticated account the same BDSPro tokens and session.
func IssueSession(ctx context.Context, store db.Store, maker token.Maker, config util.Config, account db.Account, userAgent, clientIP string) (LoginResult, error) {
	var result LoginResult
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
