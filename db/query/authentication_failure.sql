-- name: InsertAuthenticationFailure :exec
INSERT INTO authentication_failures (account_id, method, reason, credential_mac, ip_address)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteExpiredAuthenticationFailures :execrows
WITH expired AS (
    SELECT id FROM authentication_failures
    WHERE occurred_at < sqlc.arg(cutoff)::timestamptz
    ORDER BY occurred_at, id
    LIMIT sqlc.arg(take)::integer
    FOR UPDATE SKIP LOCKED
)
DELETE FROM authentication_failures AS failure
USING expired
WHERE failure.id = expired.id;
