package oauth

import (
	"context"
	"fmt"
	"time"


	db "github.com/datvtph41107/bdspro/models/db"
)

// OAuthCleanupCounts records a bounded, committed sweep; counts are returned
// only after every DELETE in the sweep commits.
type OAuthCleanupCounts struct {
	AuthCodes     int64
	RefreshTokens int64
	AccessTokens  int64
}

// Each invocation deletes at most this many rows per OAuth table. Operators
// must schedule additional sweeps and monitor the expiry backlog.
const oauthCleanupBatch = 2000

// CleanupExpiredOAuth deletes only expired OAuth evidence, preserving live
// refresh-token rotation chains. The schema intentionally has a refresh ->
// access ON DELETE CASCADE FK: delete expired refresh rows first, then delete
// an expired access row ONLY if no refresh row still references it.
// Expired/consumed refresh rows remain until expiry to allow replay detection.
func CleanupExpiredOAuth(ctx context.Context, cutoff time.Time) (OAuthCleanupCounts, error) {
	if cutoff.IsZero() {
		return OAuthCleanupCounts{}, fmt.Errorf("OAuth cleanup requires a nonzero cutoff")
	}
	var result OAuthCleanupCounts
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		tx, err := db.GetEngine(txCtx)
		if err != nil {
			return err
		}
		ctx := txCtx
		codes, err := tx.Exec(ctx, `WITH expired AS (
			SELECT code_hash FROM oauth_auth_codes
			WHERE expires_at <= $1
			ORDER BY expires_at, code_hash
			LIMIT $2 FOR UPDATE SKIP LOCKED
		) DELETE FROM oauth_auth_codes AS a
		USING expired AS e WHERE a.code_hash = e.code_hash`, cutoff, oauthCleanupBatch)
		if err != nil {
			return fmt.Errorf("delete expired OAuth codes: %w", err)
		}
		result.AuthCodes = codes.RowsAffected()

		refresh, err := tx.Exec(ctx, `WITH expired AS (
			SELECT token_hash FROM oauth_refresh_tokens
			WHERE expires_at <= $1
			ORDER BY expires_at, token_hash
			LIMIT $2 FOR UPDATE SKIP LOCKED
		) DELETE FROM oauth_refresh_tokens AS r
		USING expired AS e WHERE r.token_hash = e.token_hash`, cutoff, oauthCleanupBatch)
		if err != nil {
			return fmt.Errorf("delete expired OAuth refresh tokens: %w", err)
		}
		result.RefreshTokens = refresh.RowsAffected()

		access, err := tx.Exec(ctx, `WITH expired AS (
			SELECT a.token_hash FROM oauth_access_tokens AS a
			WHERE a.expires_at <= $1
			AND NOT EXISTS (
				SELECT 1 FROM oauth_refresh_tokens AS r
				WHERE r.access_token_hash = a.token_hash
			)
			ORDER BY a.expires_at, a.token_hash
			LIMIT $2 FOR UPDATE OF a SKIP LOCKED
		) DELETE FROM oauth_access_tokens AS a
		USING expired AS e WHERE a.token_hash = e.token_hash`, cutoff, oauthCleanupBatch)
		if err != nil {
			return fmt.Errorf("delete unreferenced expired OAuth access tokens: %w", err)
		}
		result.AccessTokens = access.RowsAffected()
		return nil
	})
	if err != nil {
		return OAuthCleanupCounts{}, err
	}
	return result, nil
}
