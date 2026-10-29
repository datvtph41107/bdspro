-- name: ListActiveSessions :many
SELECT id, user_agent, client_ip, created_at, expires_at
FROM sessions
WHERE account_id = $1 AND is_blocked = FALSE AND expires_at > now()
ORDER BY created_at DESC, id;

-- name: RevokeOwnedSession :execrows
UPDATE sessions SET is_blocked = TRUE
WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id)
  AND is_blocked = FALSE AND expires_at > now();

-- name: RevokeAllAccountSessions :execrows
UPDATE sessions SET is_blocked = TRUE
WHERE account_id = $1 AND is_blocked = FALSE AND expires_at > now();
