-- name: GetPhoneAccountIDQuery :one
SELECT account_id FROM phone_credentials WHERE phone_e164 = $1;

-- name: UpsertPhoneChallengeQuery :execrows
INSERT INTO phone_otp_challenges
    (phone_e164, purpose, account_id, code_mac, dev_code, expires_at, next_send_at,
     failed_attempts, attempt_window_started_at, locked_until)
VALUES (
    sqlc.arg(phone_e164), sqlc.arg(purpose), sqlc.arg(account_id),
    sqlc.arg(code_mac), NULLIF(sqlc.arg(dev_code)::text, ''),
    now() + interval '5 minutes', now() + interval '60 seconds',
    0, now(), 'epoch'::timestamptz
)
ON CONFLICT (phone_e164, purpose) DO UPDATE
SET account_id = EXCLUDED.account_id,
    code_mac = EXCLUDED.code_mac,
    dev_code = EXCLUDED.dev_code,
    expires_at = EXCLUDED.expires_at,
    next_send_at = EXCLUDED.next_send_at,
    failed_attempts = CASE
        WHEN phone_otp_challenges.attempt_window_started_at <= now() - interval '15 minutes'
        THEN 0 ELSE phone_otp_challenges.failed_attempts END,
    attempt_window_started_at = CASE
        WHEN phone_otp_challenges.attempt_window_started_at <= now() - interval '15 minutes'
        THEN now() ELSE phone_otp_challenges.attempt_window_started_at END,
    locked_until = CASE
        WHEN phone_otp_challenges.attempt_window_started_at <= now() - interval '15 minutes'
        THEN 'epoch'::timestamptz ELSE phone_otp_challenges.locked_until END
WHERE phone_otp_challenges.next_send_at <= now()
  AND phone_otp_challenges.locked_until <= now()
  AND (phone_otp_challenges.failed_attempts < 5
       OR phone_otp_challenges.attempt_window_started_at <= now() - interval '15 minutes');

-- name: LockPhoneChallengeQuery :one
SELECT account_id, code_mac, expires_at, failed_attempts, locked_until
FROM phone_otp_challenges
WHERE phone_e164 = sqlc.arg(phone_e164)
  AND purpose = sqlc.arg(purpose)
FOR UPDATE;

-- name: RecordPhoneChallengeFailureQuery :exec
UPDATE phone_otp_challenges
SET failed_attempts = failed_attempts + 1,
    locked_until = CASE
        WHEN failed_attempts + 1 >= 5 THEN now() + interval '15 minutes'
        ELSE locked_until END
WHERE phone_e164 = sqlc.arg(phone_e164)
  AND purpose = sqlc.arg(purpose);

-- name: BindPhoneCredentialQuery :exec
INSERT INTO phone_credentials (account_id, phone_e164, verified_at)
VALUES (sqlc.arg(account_id), sqlc.arg(phone_e164), now())


-- name: LockPhoneCredentialQuery :one
SELECT account_id FROM phone_credentials WHERE phone_e164 = $1 FOR SHARE;

-- name: DeletePhoneChallengeQuery :exec
DELETE FROM phone_otp_challenges
WHERE phone_e164 = sqlc.arg(phone_e164)
  AND purpose = sqlc.arg(purpose);
