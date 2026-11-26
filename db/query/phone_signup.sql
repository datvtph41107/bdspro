-- name: CreatePhoneOnlyAccount :one
INSERT INTO accounts (person_id, email) VALUES (sqlc.arg(person_id), NULL)
RETURNING id, person_id, COALESCE(email, '')::text AS email, created_at, is_email_verified;

-- name: UpsertPhoneSignup :execrows
INSERT INTO phone_signup_challenges
    (phone_e164, code_mac, dev_code, expires_at, next_send_at, failed_attempts, locked_until)
VALUES (sqlc.arg(phone_e164), sqlc.arg(code_mac), NULLIF(sqlc.arg(dev_code)::text, ''),
        now() + interval '5 minutes', now() + interval '60 seconds', 0, 'epoch'::timestamptz)
ON CONFLICT(phone_e164) DO UPDATE SET
 code_mac=EXCLUDED.code_mac, dev_code=EXCLUDED.dev_code,
 expires_at=EXCLUDED.expires_at, next_send_at=EXCLUDED.next_send_at,
 failed_attempts=0, locked_until='epoch'::timestamptz
WHERE phone_signup_challenges.next_send_at <= now()
 AND phone_signup_challenges.locked_until <= now();

-- name: LockPhoneSignup :one
SELECT code_mac, expires_at, failed_attempts, locked_until
FROM phone_signup_challenges WHERE phone_e164 = $1 FOR UPDATE;

-- name: FailPhoneSignup :exec
UPDATE phone_signup_challenges
SET failed_attempts = failed_attempts + 1,
 locked_until = CASE WHEN failed_attempts + 1 >= 5
 THEN now() + interval '15 minutes' ELSE locked_until END
WHERE phone_e164 = $1;

-- name: DeletePhoneSignup :exec
DELETE FROM phone_signup_challenges WHERE phone_e164 = $1;

-- name: CreatePhoneContinuation :exec
INSERT INTO phone_continuation_challenges(id,phone_e164,browser_digest,code_mac,expires_at,resend_after)
VALUES (sqlc.arg(id)::uuid,sqlc.arg(phone_e164)::text,sqlc.arg(browser_digest)::bytea,sqlc.arg(code_mac)::bytea,clock_timestamp()+interval '5 minutes',clock_timestamp()+interval '60 seconds');

-- name: ReadPhoneContinuationDestination :one
SELECT phone_e164 FROM phone_continuation_challenges
WHERE id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea;

-- name: LockPhoneContinuation :one
SELECT id,phone_e164,code_mac,failed_attempts,expires_at
FROM phone_continuation_challenges
WHERE id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest IS NULL
FOR UPDATE;

-- name: RotatePhoneContinuation :execrows
UPDATE phone_continuation_challenges SET code_mac=sqlc.arg(code_mac)::bytea,
    expires_at=clock_timestamp()+interval '5 minutes',resend_after=clock_timestamp()+interval '60 seconds'
WHERE id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5
    AND resend_after<=clock_timestamp() AND created_at>clock_timestamp()-interval '30 minutes';

-- name: FailPhoneContinuation :exec
UPDATE phone_continuation_challenges SET failed_attempts=failed_attempts+1
WHERE id=$1 AND failed_attempts<5 AND proof_digest IS NULL;

-- name: VerifyPhoneContinuation :execrows
UPDATE phone_continuation_challenges SET code_mac=NULL,proof_digest=sqlc.arg(proof_digest)::bytea,
    expires_at=clock_timestamp()+interval '15 minutes'
WHERE id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5 AND expires_at>clock_timestamp();

-- name: LockPhoneContinuationProof :one
SELECT id,phone_e164 FROM phone_continuation_challenges
WHERE proof_digest=sqlc.arg(proof_digest)::bytea AND browser_digest=sqlc.arg(browser_digest)::bytea
AND expires_at>clock_timestamp() FOR UPDATE;

-- name: DeletePhoneContinuation :exec
DELETE FROM phone_continuation_challenges WHERE id=$1;

-- name: DeleteExpiredPhoneContinuations :execrows
DELETE FROM phone_continuation_challenges WHERE id IN (
    SELECT id FROM phone_continuation_challenges
    WHERE expires_at<clock_timestamp()-interval '30 minutes'
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);
