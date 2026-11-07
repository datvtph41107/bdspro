package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// LockAccountForCredentialRecovery serializes a reset with session inserts
// (accounts FK KEY SHARE) and OAuth writes (explicit accounts FOR SHARE).
// Caller holds this lock through the final commit or rollback.
func LockAccountForCredentialRecovery(ctx context.Context, tx pgx.Tx, accountID int64) error {
	var id int64
	return tx.QueryRow(ctx, `SELECT id FROM accounts WHERE id=$1 FOR UPDATE`, accountID).Scan(&id)
}

// RevokeOAuthForCredentialRecovery must be called in the transaction holding
// the account FOR UPDATE lock. Do not revoke OAuth clients; they can belong
// to a different user and remain registered for their other authorizations.
func RevokeOAuthForCredentialRecovery(ctx context.Context, tx pgx.Tx, accountID int64) error {
	if _, err := tx.Exec(ctx, `UPDATE oauth_grants
		SET revoked_at=COALESCE(revoked_at,now()),version=version+1,updated_at=now()
		WHERE account_id=$1`, accountID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM oauth_auth_codes
		WHERE account_id=$1 AND used_at IS NULL`, accountID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET revoked_at=now()
		WHERE account_id=$1 AND revoked_at IS NULL`, accountID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE oauth_access_tokens SET revoked_at=now()
		WHERE account_id=$1 AND revoked_at IS NULL`, accountID)
	return err
}
