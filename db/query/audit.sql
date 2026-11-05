-- name: DeleteOldAuditEvents :execrows
WITH expired AS (
    SELECT id FROM audit_events
    WHERE created_at < sqlc.arg(cutoff)::timestamptz
    ORDER BY created_at, id
    LIMIT sqlc.arg(take)::integer
    FOR UPDATE SKIP LOCKED
)
DELETE FROM audit_events AS e
USING expired
WHERE e.id = expired.id;

-- name: InsertAuditEvent :exec
INSERT INTO audit_events (account_id, action, method, reason, subject_mac, ip_address)
VALUES ($1, 'auth.login.rejected', $2, $3, $4, $5);
