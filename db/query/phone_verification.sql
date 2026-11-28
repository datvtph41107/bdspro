-- name: CreatePhoneVerification :exec
INSERT INTO verifications(kind,id,destination,browser_digest,code_mac,expires_at,resend_after)
VALUES ('phone',sqlc.arg(id)::uuid,sqlc.arg(phone_e164)::text,sqlc.arg(browser_digest)::bytea,sqlc.arg(code_mac)::bytea,clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),clock_timestamp()+(sqlc.arg(cooldown_seconds)::integer * interval '1 second'));

-- name: ReadPhoneVerificationDestination :one
SELECT destination AS phone_e164 FROM verifications
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea;

-- name: LockPhoneVerification :one
SELECT id,destination AS phone_e164,code_mac,failed_attempts,expires_at
FROM verifications
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest IS NULL
FOR UPDATE;

-- name: RotatePhoneVerification :execrows
UPDATE verifications SET code_mac=sqlc.arg(code_mac)::bytea,
    expires_at=clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),resend_after=clock_timestamp()+(sqlc.arg(cooldown_seconds)::integer * interval '1 second')
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<sqlc.arg(max_failed_attempts)::integer
    AND resend_after<=clock_timestamp() AND created_at>clock_timestamp()-(sqlc.arg(resend_horizon_minutes)::integer * interval '1 minute');

-- name: FailPhoneVerification :exec
UPDATE verifications SET failed_attempts=failed_attempts+1
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND failed_attempts<sqlc.arg(max_failed_attempts)::integer AND proof_digest IS NULL;

-- name: VerifyPhoneVerification :execrows
UPDATE verifications SET code_mac=NULL,proof_digest=sqlc.arg(proof_digest)::bytea,
    expires_at=clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute')
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<sqlc.arg(max_failed_attempts)::integer AND expires_at>clock_timestamp();

-- name: LockPhoneVerificationProof :one
WITH locked_proof AS MATERIALIZED (
    SELECT id,destination,expires_at FROM verifications
    WHERE kind='phone' AND proof_digest=sqlc.arg(proof_digest)::bytea AND browser_digest=sqlc.arg(browser_digest)::bytea
    FOR UPDATE
)
SELECT id,destination AS phone_e164 FROM locked_proof
WHERE expires_at>clock_timestamp();

-- name: DeletePhoneVerification :exec
DELETE FROM verifications WHERE kind='phone' AND id=$1;

-- name: DeleteExpiredPhoneVerifications :execrows
DELETE FROM verifications WHERE kind='phone' AND id IN (
    SELECT id FROM verifications
    WHERE kind='phone' AND expires_at<clock_timestamp()-(sqlc.arg(retention_minutes)::integer * interval '1 minute')
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);
