package auth

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidCredentials = util.WithPublic(util.ErrUnauthenticated, "invalid password proof", "invalid_credentials", "invalid email, username or password")

type LoginResult struct {
	AccessToken string                `json:"access_token"`
	TokenType   string                `json:"token_type"`
	ExpiresIn   int64                 `json:"expires_in"`
	Status      string                `json:"status"`
	User        authmodel.CurrentUser `json:"user"`
	Refresh     string                `json:"-"`
	ExpiresAt   time.Time             `json:"-"`
	Remember    bool                  `json:"-"`
}

func LoginPassword(ctx context.Context, identifier, password, clientIP, userAgent string, remember bool) (LoginResult, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return LoginResult{}, util.WithOperation("auth.login.open_queries", err)
	}
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	var accountID int64
	if strings.Contains(identifier, "@") {
		a, lookupErr := q.ReadAccountByEmail(ctx, identifier)
		accountID, err = a.ID, lookupErr
	} else {
		accountID, err = q.ReadAccountIDByUsername(ctx, identifier)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, rejectMissingPassword(password)
	}
	if err != nil {
		return LoginResult{}, util.WithOperation("auth.login.lookup_account", err)
	}
	hash, err := q.GetPasswordHash(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, rejectMissingPassword(password)
	}
	if err != nil {
		return LoginResult{}, util.WithOperation("auth.login.read_password", err)
	}
	if err = token.CheckPassword(password, hash); err != nil {
		if errors.Is(err, token.ErrPasswordBusy) {
			return LoginResult{}, err
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	result, err := db.WithTx2(ctx, func(txCtx context.Context) (LoginResult, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return LoginResult{}, err
		}
		if _, err = q.LockSessionRevocationAccount(txCtx, accountID); errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		} else if err != nil {
			return LoginResult{}, err
		}
		current, err := q.ReadAccountByID(txCtx, accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		if err != nil {
			return LoginResult{}, err
		}
		currentHash, err := q.GetPasswordHash(txCtx, accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		if err != nil {
			return LoginResult{}, err
		}
		identity, err := q.GetAccountIdentity(txCtx, accountID)
		if err != nil {
			return LoginResult{}, err
		}
		matches := identity.Username == identifier
		if strings.Contains(identifier, "@") {
			matches = strings.ToLower(strings.TrimSpace(current.Email)) == identifier
		}
		if identity.Status != "active" || !current.IsEmailVerified || !matches || currentHash != hash {
			return LoginResult{}, ErrInvalidCredentials
		}
		return issueSession(txCtx, q, accountID, clientIP, userAgent, remember)
	})
	return result, util.WithOperation("auth.login.transaction", err)
}

func rejectMissingPassword(password string) error {
	if err := token.CheckMissingPassword(password); errors.Is(err, token.ErrPasswordBusy) {
		return err
	}
	return ErrInvalidCredentials
}

func issueSession(ctx context.Context, q *db.Queries, accountID int64, clientIP, userAgent string, remember bool) (LoginResult, error) {
	if !db.InTransaction(ctx) {
		return LoginResult{}, errors.New("session issuance requires transaction")
	}
	if _, err := netip.ParseAddr(clientIP); err != nil {
		return LoginResult{}, err
	}
	userAgent = strings.ToValidUTF8(userAgent, "")
	if len(userAgent) > 2048 {
		userAgent = userAgent[:2048]
		for !utf8.ValidString(userAgent) {
			userAgent = userAgent[:len(userAgent)-1]
		}
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return LoginResult{}, err
	}
	ttl := setting.Security.RefreshTokenDuration
	if remember {
		ttl = setting.Security.RememberSessionDuration
	}
	expires := time.Now().UTC().Add(ttl)
	raw, err := token.NewRefreshCredential(setting.Security.TokenSymmetricKey, id, remember)
	if err != nil {
		return LoginResult{}, err
	}
	row, err := q.CreateSession(ctx, db.CreateSessionParams{
		ID: id, AccountID: accountID, RefreshToken: token.RefreshDigest(raw),
		UserAgent: userAgent, ClientIp: clientIP, ExpiresAt: expires,
	})
	if err != nil {
		return LoginResult{}, util.WithOperation("auth.session.create", err)
	}
	result, err := sessionResult(ctx, row.AccountID, row.ID, row.ExpiresAt, raw, remember)
	if err != nil {
		return LoginResult{}, util.WithOperation("auth.session.result", err)
	}
	if err = q.CreateAccountConnectionHistory(ctx, db.CreateAccountConnectionHistoryParams{
		AccountID: accountID, SessionID: id, IpAddress: clientIP, UserAgent: userAgent,
	}); err != nil {
		return LoginResult{}, util.WithOperation("auth.session.connection_history", err)
	}
	return result, nil
}

func sessionResult(ctx context.Context, accountID int64, id uuid.UUID, expiry time.Time, raw string, remember bool) (LoginResult, error) {
	access, payload, err := token.ProcessMaker().CreateSessionAccessToken(accountID, setting.Security.AccessTokenDuration, id, expiry)
	if errors.Is(err, token.ErrExpiredToken) {
		return LoginResult{}, util.ErrUnauthenticated
	}
	if err != nil {
		return LoginResult{}, err
	}
	u, err := GetCurrentUser(ctx, accountID)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Status: "authenticated", AccessToken: access, TokenType: "Bearer",
		ExpiresIn: int64(payload.ExpiredAt.Sub(payload.IssuedAt) / time.Second), User: u, Refresh: raw, ExpiresAt: expiry, Remember: remember,
	}, nil
}
