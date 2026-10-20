-- name: CreateLoginSession :one
INSERT INTO sessions (account_id, token_hash, expires_at)
VALUES (sqlc.arg(account_id), sqlc.arg(token_hash), now() + interval '24 hours')
RETURNING expires_at;

-- name: GetSessionAccount :one
SELECT a.id FROM sessions se JOIN accounts a ON a.id = se.account_id
WHERE se.token_hash = $1 AND se.revoked_at IS NULL AND se.expires_at > now()
  AND a.closed_at IS NULL;

-- name: LockSessionAccount :one
-- All operations that change account-wide session state must lock this row.
SELECT id FROM accounts WHERE id = $1 AND closed_at IS NULL FOR UPDATE;

-- name: PruneExcessSessions :exec
-- Keep the five most recently created active sessions; ties break by id.
WITH excess AS (
    SELECT s.id FROM sessions AS s
    WHERE s.account_id = sqlc.arg(account_id)::bigint AND s.revoked_at IS NULL AND s.expires_at > now()
    ORDER BY s.created_at DESC, s.id DESC OFFSET 5
)
UPDATE sessions AS dst SET revoked_at = now()
WHERE dst.id IN (SELECT id FROM excess);

-- name: RevokeAccountSessions :exec
UPDATE sessions SET revoked_at = now()
WHERE account_id = $1 AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL;
