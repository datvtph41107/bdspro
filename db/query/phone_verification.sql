-- name: CreatePhoneVerification :exec
INSERT INTO verifications(kind,id,destination,browser_digest,code_mac,expires_at,resend_after)
VALUES ('phone',sqlc.arg(id)::uuid,sqlc.arg(phone_e164)::text,sqlc.arg(browser_digest)::bytea,sqlc.arg(code_mac)::bytea,clock_timestamp()+interval '5 minutes',clock_timestamp()+interval '60 seconds');

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
    expires_at=clock_timestamp()+interval '5 minutes',resend_after=clock_timestamp()+interval '60 seconds'
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5
    AND resend_after<=clock_timestamp() AND created_at>clock_timestamp()-interval '30 minutes';

-- name: FailPhoneVerification :exec
UPDATE verifications SET failed_attempts=failed_attempts+1
WHERE kind='phone' AND id=$1 AND failed_attempts<5 AND proof_digest IS NULL;

-- name: VerifyPhoneVerification :execrows
UPDATE verifications SET code_mac=NULL,proof_digest=sqlc.arg(proof_digest)::bytea,
    expires_at=clock_timestamp()+interval '15 minutes'
WHERE kind='phone' AND id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5 AND expires_at>clock_timestamp();

-- name: LockPhoneVerificationProof :one
SELECT id,destination AS phone_e164 FROM verifications
WHERE kind='phone' AND proof_digest=sqlc.arg(proof_digest)::bytea AND browser_digest=sqlc.arg(browser_digest)::bytea
AND expires_at>clock_timestamp() FOR UPDATE;

-- name: DeletePhoneVerification :exec
DELETE FROM verifications WHERE kind='phone' AND id=$1;

-- name: DeleteExpiredPhoneVerifications :execrows
DELETE FROM verifications WHERE kind='phone' AND id IN (
    SELECT id FROM verifications
    WHERE kind='phone' AND expires_at<clock_timestamp()-interval '30 minutes'
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);
