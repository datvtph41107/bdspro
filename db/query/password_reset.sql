-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash, account_id, expires_at)
VALUES (sqlc.arg(token_hash), sqlc.arg(account_id), now() + interval '15 minutes');

-- name: IssuePasswordResetTokenIfCurrentVersion :execrows
-- FOR SHARE serializes issuance with recovery's accounts FOR UPDATE.
-- The returning row count is zero for stale or unverified intents.
INSERT INTO password_reset_tokens (token_hash, account_id, expires_at)
SELECT sqlc.arg(token_hash)::bytea, id, now() + interval '15 minutes'
FROM accounts
WHERE id = sqlc.arg(account_id)::bigint
  AND password_reset_version = sqlc.arg(reset_version)::bigint
  AND is_email_verified
FOR SHARE;

-- name: BumpPasswordResetVersion :exec
-- Called after consuming a reset proof while holding accounts FOR UPDATE.
UPDATE accounts
SET password_reset_version = password_reset_version + 1
WHERE id = $1;

-- name: ConsumePasswordResetToken :one
DELETE FROM password_reset_tokens
WHERE token_hash = sqlc.arg(token_hash) AND expires_at > now()
RETURNING account_id;

-- name: DeletePasswordResetTokensForAccount :exec
DELETE FROM password_reset_tokens WHERE account_id = $1;

-- name: UpsertPasswordCredential :exec
INSERT INTO password_credentials (account_id, password_hash)
VALUES (sqlc.arg(account_id), sqlc.arg(password_hash))
ON CONFLICT (account_id) DO UPDATE
SET password_hash = EXCLUDED.password_hash, updated_at = now();

-- name: DeleteSessionsForAccount :exec
DELETE FROM sessions WHERE account_id = $1;
