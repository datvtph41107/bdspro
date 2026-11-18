-- name: InsertAuditEvent :exec
INSERT INTO audit_event (
    action,
    actor_id,
    actor_name,
    actor_credential,
    impersonator_id,
    impersonator_name,
    scope_type,
    scope_ref,
    scope_name,
    target_type,
    target_ref,
    origin,
    message,
    metadata,
    ip_address,
    subject_account_id,
    subject_mac
)
VALUES (
    sqlc.arg(action)::text,
    sqlc.narg(actor_id)::bigint,
    sqlc.arg(actor_name)::text,
    sqlc.arg(actor_credential)::text,
    sqlc.narg(impersonator_id)::bigint,
    sqlc.arg(impersonator_name)::text,
    sqlc.arg(scope_type)::text,
    sqlc.arg(scope_ref)::text,
    sqlc.arg(scope_name)::text,
    sqlc.arg(target_type)::text,
    sqlc.arg(target_ref)::text,
    sqlc.arg(origin)::text,
    sqlc.arg(message)::text,
    sqlc.arg(metadata)::jsonb,
    NULLIF(sqlc.arg(ip_address)::text, '')::inet,
    sqlc.narg(subject_account_id)::bigint,
    sqlc.narg(subject_mac)::bytea
);

-- name: ListAuditEvents :many
SELECT *
FROM audit_event
WHERE
    (sqlc.narg(actor_id)::bigint IS NULL
        OR actor_id = sqlc.narg(actor_id))
    AND
    (sqlc.narg(action)::text IS NULL
        OR action = sqlc.narg(action))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(take)::integer
OFFSET sqlc.arg(skip)::integer;

-- name: DeleteOldAuditEvents :execrows
WITH expired AS (
    SELECT id
    FROM audit_event
    WHERE created_at < sqlc.arg(cutoff)::timestamptz
    ORDER BY created_at, id
    LIMIT sqlc.arg(take)::integer
    FOR UPDATE SKIP LOCKED
)
DELETE FROM audit_event AS e
USING expired
WHERE e.id = expired.id;
