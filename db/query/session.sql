-- name: CreateSession :one
INSERT INTO sessions (id, account_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, account_id, refresh_token, user_agent, client_ip, is_blocked, created_at, expires_at;

-- name: GetSession :one
SELECT id, account_id, refresh_token, user_agent, client_ip, is_blocked, created_at, expires_at
FROM sessions WHERE id = $1 LIMIT 1;

-- name: BlockSession :one
UPDATE sessions
SET is_blocked = TRUE
WHERE id = $1
  AND account_id = $2
  AND refresh_token = $3
  AND is_blocked = FALSE
  AND expires_at > now()
RETURNING id, account_id, refresh_token, user_agent, client_ip, is_blocked, created_at, expires_at;
