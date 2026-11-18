-- name: CreateNotification :execrows
INSERT INTO notification (
    account_id,
    event_type,
    source_type,
    source_ref,
    updated_by_account_id,
    payload,
    dedupe_key
)
VALUES (
    sqlc.arg(account_id)::bigint,
    sqlc.arg(event_type)::text,
    sqlc.arg(source_type)::text,
    sqlc.arg(source_ref)::text,
    sqlc.narg(updated_by_account_id)::bigint,
    sqlc.arg(payload)::jsonb,
    sqlc.narg(dedupe_key)::text
)
ON CONFLICT (account_id, dedupe_key)
WHERE dedupe_key IS NOT NULL
DO NOTHING;

-- name: ListAccountNotifications :many
SELECT *
FROM notification
WHERE account_id = sqlc.arg(account_id)::bigint
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg(take)::integer
OFFSET sqlc.arg(skip)::integer;

-- name: CountUnreadNotifications :one
SELECT COUNT(*)::bigint
FROM notification
WHERE account_id = sqlc.arg(account_id)::bigint
  AND status = 1;

-- name: MarkNotificationRead :execrows
UPDATE notification
SET status = 2,
    read_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(notification_id)::bigint
  AND account_id = sqlc.arg(account_id)::bigint
  AND status = 1;

-- name: MarkAllNotificationsRead :execrows
UPDATE notification
SET status = 2,
    read_at = now(),
    updated_at = now()
WHERE account_id = sqlc.arg(account_id)::bigint
  AND status = 1;

-- name: DeleteNotification :execrows
DELETE FROM notification
WHERE id = sqlc.arg(notification_id)::bigint
  AND account_id = sqlc.arg(account_id)::bigint;
