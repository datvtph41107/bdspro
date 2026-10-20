-- name: GetVerifiedPhoneLoginTarget :one
-- A phone login cannot target an unverified number or closed account.
SELECT ap.id AS account_phone_id, a.id AS account_id
FROM account_phones ap
JOIN accounts a ON a.id = ap.account_id
WHERE ap.phone = $1 AND ap.verified_at IS NOT NULL AND a.closed_at IS NULL;

-- name: CanIssuePhoneLogin :one
-- Enrollment and login share the same 60-second/daily destination SMS budget.
-- Account-level login quota bounds repeated sends across different numbers.
SELECT (
  (
    (SELECT count(*) FROM phone_login_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text AND c.created_at > now() - interval '60 seconds')
    +
    (SELECT count(*) FROM phone_verification_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text AND c.created_at > now() - interval '60 seconds')
  ) = 0
  AND
  (
    (SELECT count(*) FROM phone_login_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text AND c.created_at > now() - interval '24 hours')
    +
    (SELECT count(*) FROM phone_verification_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text AND c.created_at > now() - interval '24 hours')
  ) < 5
  AND
  (SELECT count(*) FROM phone_login_challenges c
   JOIN account_phones p ON p.id = c.account_phone_id
   WHERE p.account_id = sqlc.arg(account_id)::bigint
     AND c.created_at > now() - interval '24 hours') < 10
) AS allowed;

-- name: InvalidatePriorPhoneLoginChallenges :exec
UPDATE phone_login_challenges SET consumed_at = now()
WHERE account_phone_id = $1 AND consumed_at IS NULL;

-- name: CreatePhoneLoginChallenge :exec
INSERT INTO phone_login_challenges(account_phone_id, challenge_hash, nonce, otp_mac)
VALUES (sqlc.arg(account_phone_id), sqlc.arg(challenge_hash),
        sqlc.arg(nonce), sqlc.arg(otp_mac));

-- name: GetPhoneLoginChallengeAccount :one
-- The first read only chooses an account lock; it is not authorization.
SELECT ap.account_id
FROM phone_login_challenges c
JOIN account_phones ap ON ap.id = c.account_phone_id
WHERE c.challenge_hash = $1;

-- name: LockActivePhoneLoginChallenge :one
-- Re-evaluate all eligibility under the account lock and row lock.
SELECT c.id, c.account_phone_id, c.nonce, c.otp_mac
FROM phone_login_challenges c
JOIN account_phones ap ON ap.id = c.account_phone_id
JOIN accounts a ON a.id = ap.account_id
WHERE c.challenge_hash = sqlc.arg(challenge_hash)
  AND ap.account_id = sqlc.arg(account_id)
  AND ap.verified_at IS NOT NULL AND a.closed_at IS NULL
  AND c.consumed_at IS NULL AND c.expires_at > now() AND c.attempts < 5
FOR UPDATE OF c;

-- name: CountPhoneLoginAttempt :exec
UPDATE phone_login_challenges SET attempts = attempts + 1
WHERE id = $1 AND consumed_at IS NULL AND attempts < 5;

-- name: ConsumePhoneLoginChallenge :exec
UPDATE phone_login_challenges SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL;
