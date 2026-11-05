-- name: CreateAccountConnectionHistory :exec
INSERT INTO account_connection_histories (account_id, session_id, action, ip_address, user_agent)
VALUES ($1, $2, 'login', $3, $4);

-- name: DeleteExpiredAccountConnectionHistories :execrows
WITH expired AS (
    SELECT id FROM account_connection_histories
    WHERE created_at < sqlc.arg(cutoff)::timestamptz
    ORDER BY created_at, id
    LIMIT sqlc.arg(take)::integer
    FOR UPDATE SKIP LOCKED
)
DELETE FROM account_connection_histories AS history
USING expired
WHERE history.id = expired.id;

-- name: ListAccountConnectionHistories :many
SELECT id, session_id, action, ip_address, user_agent, created_at
FROM account_connection_histories
WHERE account_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 50;
