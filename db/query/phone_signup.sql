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
