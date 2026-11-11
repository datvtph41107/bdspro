package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

// oauthAuthorizationKey is a namespaced, deterministic advisory-lock key.
// Collisions only add serialization; they never grant authorization.
func oauthAuthorizationKey(accountID int64, clientID uuid.UUID) int64 {
	var account [8]byte
	binary.BigEndian.PutUint64(account[:], uint64(accountID))
	h := sha256.New()
	_, _ = h.Write([]byte("oauth.authorization.account_client.v1"))
	_, _ = h.Write(account[:])
	_, _ = h.Write(clientID[:])
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// All token/code issuance for the same account/client pair must acquire this
// lock before any row lock; refreshing also acquires the family lock second.
func lockOAuthAuthorization(ctx context.Context, tx db.DBTX, accountID int64, clientID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1::bigint)`, oauthAuthorizationKey(accountID, clientID))
	if err != nil {
		return err
	}
	// Password recovery locks the account row FOR UPDATE before revoking
	// grants. OAuth writers lock it FOR SHARE after their pair advisory lock.
	// This serializes recovery against code issuance and refresh rotation.
	var id int64
	return tx.QueryRow(ctx, `SELECT id FROM accounts WHERE id=$1 FOR SHARE`, accountID).Scan(&id)
}

// ListOAuthAuthorizations lists active, persistently approved grants even if
// their access tokens have expired. Application owner != authorizing account.
func ListOAuthAuthorizations(ctx context.Context, accountID int64) ([]OAuthAuthorizedClient, error) {
	dbx, resolveErr := db.GetEngine(ctx)
	if resolveErr != nil {
		return nil, resolveErr
	}
	rows, err := dbx.Query(ctx, `SELECT c.id,c.name FROM oauth_grants g
        JOIN oauth_clients c ON c.id=g.client_id
        WHERE g.account_id=$1 AND g.revoked_at IS NULL AND c.revoked_at IS NULL
        ORDER BY g.approved_at DESC,c.id LIMIT 100`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OAuthAuthorizedClient, 0)
	for rows.Next() {
		var entry OAuthAuthorizedClient
		if err := rows.Scan(&entry.ClientID, &entry.Name); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

// RevokeOAuthAuthorization revokes all live credentials and unused codes for
// THIS account/client pair. It never revokes the application or other users.
// Idempotent by design: unknown and already-revoked pairs return nil.
func RevokeOAuthAuthorization(ctx context.Context, accountID int64, clientID uuid.UUID) error {
	return db.WithTx(ctx, func(txCtx context.Context) error {
		tx, err := db.GetEngine(txCtx)
		if err != nil {
			return err
		}
		ctx := txCtx
		if err := lockOAuthAuthorization(ctx, tx, accountID, clientID); err != nil {
			return err
		}
		// Keep a versioned tombstone even when no previous grant exists.
		// This prevents an old approval preview from re-authorizing afterward.
		if _, err := tx.Exec(ctx, `INSERT INTO oauth_grants
			(account_id,client_id,scope,version,approved_at,revoked_at)
			SELECT $1,c.id,'profile:read',1,NULL,now() FROM oauth_clients c
			JOIN accounts a ON a.id=$1 WHERE c.id=$2
			ON CONFLICT (account_id,client_id) DO UPDATE SET
			version=oauth_grants.version+1,revoked_at=now(),updated_at=now()`,
			accountID, clientID); err != nil {
			return err
		}
		// No revoked_at on auth codes in the canonical schema. Unused codes
		// are removed, so an unredeemed code cannot mint tokens after revoke.
		if _, err := tx.Exec(ctx, `DELETE FROM oauth_auth_codes
			WHERE account_id=$1 AND client_id=$2 AND used_at IS NULL`, accountID, clientID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at=now()
			WHERE account_id=$1 AND client_id=$2 AND revoked_at IS NULL`, accountID, clientID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE oauth_access_tokens SET revoked_at=now()
			WHERE account_id=$1 AND client_id=$2 AND revoked_at IS NULL`, accountID, clientID)
		return err
	})
}
