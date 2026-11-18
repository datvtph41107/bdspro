package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	auditmodel "github.com/datvtph41107/bdspro/models/audit"
	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/google/uuid"
)

// SessionTokens is returned only after the session and its login history
// have committed. Neither credential should ever be written to logs.
type SessionTokens struct {
	AccountID     int64     `json:"account_id"`
	SessionID     uuid.UUID `json:"session_id"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	AccessExpiry  time.Time `json:"access_expires_at"`
	RefreshExpiry time.Time `json:"refresh_expires_at"`
}

// IssueSession is a trusted service operation: only invoke it after a
// password, phone OTP, or OIDC identity has been verified. Never expose
// accountID as an unauthenticated HTTP parameter.
//
// The refresh credential is an unpredictable opaque secret. PostgreSQL
// stores only its SHA-256 digest; the cleartext leaves this function once.
func IssueSession(
	ctx context.Context,
	accountID int64,
	userAgent, clientIP string,
	maker token.Maker,
	accessTTL, refreshTTL time.Duration,
) (SessionTokens, error) {
	if ctx == nil {
		return SessionTokens{}, errors.New("auth: nil context")
	}
	if err := ctx.Err(); err != nil {
		return SessionTokens{}, err
	}
	if maker == nil || accountID <= 0 || accessTTL <= 0 || refreshTTL <= 0 ||
		len(userAgent) > 2048 || len(clientIP) > 45 ||
		strings.ContainsAny(userAgent, "\r\n") ||
		net.ParseIP(clientIP) == nil {
		return SessionTokens{}, errors.New("auth: invalid session issuance")
	}

	sessionID, err := uuid.NewRandom()
	if err != nil {
		return SessionTokens{}, err
	}
	refresh, digest, err := newOpaqueRefreshToken()
	if err != nil {
		return SessionTokens{}, err
	}
	access, accessClaims, err := maker.CreateSessionAccessToken(
		accountID, accessTTL, sessionID,
	)
	if err != nil {
		return SessionTokens{}, fmt.Errorf("auth: sign access token: %w", err)
	}
	if accessClaims == nil || accessClaims.AccountID != accountID ||
		accessClaims.SessionID != sessionID || access == "" {
		return SessionTokens{}, errors.New("auth: access signer returned mismatched claims")
	}
	expiresAt := time.Now().UTC().Add(refreshTTL)
	tokens := SessionTokens{
		AccountID: accountID, SessionID: sessionID,
		AccessToken: access, RefreshToken: refresh,
		AccessExpiry: accessClaims.ExpiredAt, RefreshExpiry: expiresAt,
	}

	err = db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		if _, err := q.CreateSession(txCtx, db.CreateSessionParams{
			ID: sessionID, AccountID: accountID, RefreshToken: digest,
			UserAgent: userAgent, ClientIp: clientIP,
			IsBlocked: false, ExpiresAt: expiresAt,
		}); err != nil {
			return fmt.Errorf("auth: persist session: %w", err)
		}
		if err := q.CreateAccountConnectionHistory(txCtx, db.CreateAccountConnectionHistoryParams{
			AccountID: accountID, SessionID: sessionID,
			UserAgent: userAgent, IpAddress: clientIP,
		}); err != nil {
			return fmt.Errorf("auth: persist login history: %w", err)
		}
		return auditmodel.RecordSessionIssued(txCtx, accountID, sessionID)
	})
	if err != nil {
		// No credentials are returned on DB/commit failure.
		return SessionTokens{}, err
	}
	return tokens, nil
}

func newOpaqueRefreshToken() (plain, digest string, err error) {
	var secret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return "", "", fmt.Errorf("auth: generate refresh token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(secret[:])
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}
