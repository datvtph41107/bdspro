package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RefreshSession atomically consumes an opaque refresh credential and issues
// a new session with a new session ID and new refresh credential.
//
// This service accepts untrusted sessionID and refreshSecret only as a PAIR:
// knowing a session UUID alone never authenticates the caller. PostgreSQL
// compares the refresh digest as part of its conditional UPDATE and only one
// concurrent rotation can succeed. No login-history event is emitted by renew.
func RefreshSession(
	ctx context.Context,
	sessionID uuid.UUID,
	refreshSecret string,
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
	if sessionID == uuid.Nil || len(refreshSecret) != 43 ||
		maker == nil || accessTTL <= 0 || refreshTTL <= 0 ||
		len(userAgent) > 2048 || len(clientIP) > 45 ||
		strings.ContainsAny(userAgent, "\r\n") || net.ParseIP(clientIP) == nil {
		return SessionTokens{}, util.ErrUnauthenticated
	}
	sum := sha256.Sum256([]byte(refreshSecret))
	digest := hex.EncodeToString(sum[:])

	return db.WithTx2(ctx, func(txCtx context.Context) (SessionTokens, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return SessionTokens{}, err
		}

		current, err := q.GetSession(txCtx, sessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionTokens{}, util.ErrUnauthenticated
		}
		if err != nil {
			return SessionTokens{}, fmt.Errorf("auth: read refresh session: %w", err)
		}
		if current.ID != sessionID || current.AccountID <= 0 ||
			current.IsBlocked || !current.ExpiresAt.After(time.Now()) ||
			subtle.ConstantTimeCompare([]byte(digest), []byte(current.RefreshToken)) != 1 {
			return SessionTokens{}, util.ErrUnauthenticated
		}

		_, err = q.BlockSession(txCtx, db.BlockSessionParams{
			ID: sessionID, AccountID: current.AccountID, RefreshToken: digest,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionTokens{}, util.ErrUnauthenticated
		}
		if err != nil {
			return SessionTokens{}, fmt.Errorf("auth: consume refresh token: %w", err)
		}

		newID, err := uuid.NewRandom()
		if err != nil {
			return SessionTokens{}, err
		}
		plain, nextDigest, err := newOpaqueRefreshToken()
		if err != nil {
			return SessionTokens{}, err
		}
		access, claims, err := maker.CreateSessionAccessToken(current.AccountID, accessTTL, newID)
		if err != nil {
			return SessionTokens{}, fmt.Errorf("auth: sign renewed access: %w", err)
		}
		if claims == nil || claims.AccountID != current.AccountID ||
			claims.SessionID != newID || access == "" {
			return SessionTokens{}, errors.New("auth: renewed signer identity mismatch")
		}
		refreshExpiry := time.Now().UTC().Add(refreshTTL)
		if _, err := q.CreateSession(txCtx, db.CreateSessionParams{
			ID: newID, AccountID: current.AccountID, RefreshToken: nextDigest,
			UserAgent: userAgent, ClientIp: clientIP, ExpiresAt: refreshExpiry,
			IsBlocked: false,
		}); err != nil {
			return SessionTokens{}, fmt.Errorf("auth: persist renewed session: %w", err)
		}

		return SessionTokens{
			AccountID: current.AccountID,
			SessionID: newID,
			AccessToken: access,
			RefreshToken: plain,
			AccessExpiry: claims.ExpiredAt,
			RefreshExpiry: refreshExpiry,
		}, nil
	})
}
