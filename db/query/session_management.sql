-- name: LockSessionRevocationAccount :one
-- Account FOR UPDATE serializes concurrent revoke-all requests and login
-- session creation (the session FK takes KEY SHARE on accounts).
SELECT id FROM accounts
WHERE id = sqlc.arg(account_id)::bigint
FOR UPDATE;

-- name: LockActingSession :one
-- Run only inside the same transaction after LockSessionRevocationAccount.
-- Recheck the caller's session while holding its row lock until commit.
SELECT id FROM sessions
WHERE id = sqlc.arg(session_id)::uuid
  AND account_id = sqlc.arg(account_id)::bigint
  AND is_blocked = FALSE AND expires_at > clock_timestamp()
FOR UPDATE;

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
