-- name: RecordPasswordLoginEvent :exec
INSERT INTO account_login_events (account_id, outcome, client_ip, user_agent)
VALUES (sqlc.narg(account_id)::bigint, sqlc.arg(outcome), sqlc.arg(client_ip), sqlc.arg(user_agent));

-- name: ListOwnLoginEvents :many
SELECT id, outcome, client_ip, user_agent, occurred_at
FROM account_login_events
WHERE account_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT 50;

-- name: DeleteExpiredLoginEvents :execrows
DELETE FROM account_login_events
WHERE occurred_at < now() - interval '90 days';
