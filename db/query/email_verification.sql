-- name: CreateEmailRegistration :exec
INSERT INTO verifications(kind,id,destination,browser_digest,code_mac,expires_at,resend_after)
VALUES ('email',sqlc.arg(id)::uuid,sqlc.arg(email)::text,sqlc.arg(browser_digest)::bytea,sqlc.arg(code_mac)::bytea,clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),clock_timestamp()+(sqlc.arg(cooldown_seconds)::integer * interval '1 second'));

-- name: ReadEmailRegistrationDestination :one
SELECT destination AS email FROM verifications WHERE kind='email' AND id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea;

-- name: LockEmailRegistration :one
SELECT id,destination AS email,code_mac,failed_attempts,expires_at,resend_after,created_at
FROM verifications
WHERE kind='email' AND id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest IS NULL
FOR UPDATE;

-- name: RotateEmailRegistration :execrows
UPDATE verifications SET code_mac=sqlc.arg(code_mac)::bytea,
    expires_at=clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute'),resend_after=clock_timestamp()+(sqlc.arg(cooldown_seconds)::integer * interval '1 second')
WHERE kind='email' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5
    AND resend_after<=clock_timestamp() AND created_at>clock_timestamp()-interval '30 minutes';

-- name: FailEmailRegistration :exec
UPDATE verifications SET failed_attempts=failed_attempts+1
WHERE kind='email' AND id=$1 AND failed_attempts<5 AND proof_digest IS NULL;

-- name: VerifyEmailRegistration :execrows
UPDATE verifications SET code_mac=NULL,proof_digest=sqlc.arg(proof_digest)::bytea,
    expires_at=clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute')
WHERE kind='email' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5 AND expires_at>clock_timestamp();

-- name: LockEmailRegistrationProof :one
SELECT id,destination AS email FROM verifications
WHERE kind='email' AND proof_digest=sqlc.arg(proof_digest)::bytea AND browser_digest=sqlc.arg(browser_digest)::bytea
AND expires_at>clock_timestamp() FOR UPDATE;

-- name: DeleteEmailRegistration :exec
DELETE FROM verifications WHERE kind='email' AND id=$1;

-- name: DeleteExpiredEmailRegistrations :execrows
DELETE FROM verifications WHERE kind='email' AND id IN (
    SELECT id FROM verifications
    WHERE kind='email' AND expires_at<clock_timestamp()-interval '30 minutes'
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);
