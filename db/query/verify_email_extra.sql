-- name: LockVerifyEmailAccountQuery :one
SELECT is_email_verified FROM accounts WHERE id = $1 FOR UPDATE;

-- name: FindReusableVerifyEmailQuery :one
SELECT id, account_id, email, secret_code, is_used, created_at, expired_at
FROM verify_emails
WHERE account_id = sqlc.arg(account_id)
  AND email = sqlc.arg(email)
  AND is_used = FALSE
  AND expired_at > now()
ORDER BY id DESC
LIMIT 1;

-- name: CreateEmailRegistration :exec
INSERT INTO email_registration_challenges(id,email,browser_digest,code_mac,expires_at,resend_after)
VALUES (sqlc.arg(id)::uuid,sqlc.arg(email)::text,sqlc.arg(browser_digest)::bytea,sqlc.arg(code_mac)::bytea,clock_timestamp()+interval '5 minutes',clock_timestamp()+interval '60 seconds');

-- name: ReadEmailRegistrationDestination :one
SELECT email FROM email_registration_challenges WHERE id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea;

-- name: LockEmailRegistration :one
SELECT id,email,code_mac,failed_attempts,expires_at,resend_after,created_at
FROM email_registration_challenges
WHERE id=sqlc.arg(id)::uuid AND browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest IS NULL
FOR UPDATE;

-- name: RotateEmailRegistration :execrows
UPDATE email_registration_challenges SET code_mac=sqlc.arg(code_mac)::bytea,
    expires_at=clock_timestamp()+interval '5 minutes',resend_after=clock_timestamp()+interval '60 seconds'
WHERE id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5
    AND resend_after<=clock_timestamp() AND created_at>clock_timestamp()-interval '30 minutes';

-- name: FailEmailRegistration :exec
UPDATE email_registration_challenges SET failed_attempts=failed_attempts+1
WHERE id=$1 AND failed_attempts<5 AND proof_digest IS NULL;

-- name: VerifyEmailRegistration :execrows
UPDATE email_registration_challenges SET code_mac=NULL,proof_digest=sqlc.arg(proof_digest)::bytea,
    expires_at=clock_timestamp()+interval '15 minutes'
WHERE id=sqlc.arg(id)::uuid AND proof_digest IS NULL AND failed_attempts<5 AND expires_at>clock_timestamp();

-- name: LockEmailRegistrationProof :one
SELECT id,email FROM email_registration_challenges
WHERE proof_digest=sqlc.arg(proof_digest)::bytea AND browser_digest=sqlc.arg(browser_digest)::bytea
AND expires_at>clock_timestamp() FOR UPDATE;

-- name: DeleteEmailRegistration :exec
DELETE FROM email_registration_challenges WHERE id=$1;

-- name: DeleteExpiredEmailRegistrations :execrows
DELETE FROM email_registration_challenges WHERE id IN (
    SELECT id FROM email_registration_challenges
    WHERE expires_at<clock_timestamp()-interval '30 minutes'
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);
