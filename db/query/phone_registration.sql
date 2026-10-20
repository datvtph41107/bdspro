-- name: PhoneAlreadyRegistered :one
-- Closed accounts retain verified phone ownership; no silent takeover.
SELECT EXISTS (
    SELECT 1 FROM account_phones ap
    WHERE ap.phone = $1 AND ap.verified_at IS NOT NULL
) AS registered;

-- name: CanIssuePhoneRegistration :one
-- Shared destination cap across registration, login, and enrollment.
-- Call only with the same per-phone advisory lock held by other send paths.
SELECT (
    (SELECT count(*) FROM phone_registration_challenges c
      WHERE c.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '60 seconds')
    +
    (SELECT count(*) FROM phone_login_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '60 seconds')
    +
    (SELECT count(*) FROM phone_verification_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '60 seconds')
) = 0 AND (
    (SELECT count(*) FROM phone_registration_challenges c
      WHERE c.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '24 hours')
    +
    (SELECT count(*) FROM phone_login_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '24 hours')
    +
    (SELECT count(*) FROM phone_verification_challenges c
      JOIN account_phones p ON p.id = c.account_phone_id
      WHERE p.phone = sqlc.arg(phone)::text
        AND c.created_at > now() - interval '24 hours')
) < 5 AS allowed;

-- name: InvalidatePhoneRegistrationChallenges :exec
UPDATE phone_registration_challenges
SET consumed_at = now()
WHERE phone = $1 AND consumed_at IS NULL;

-- name: CreatePhoneRegistrationChallenge :exec
INSERT INTO phone_registration_challenges (phone, challenge_hash, nonce, otp_mac)
VALUES (sqlc.arg(phone), sqlc.arg(challenge_hash),
        sqlc.arg(nonce), sqlc.arg(otp_mac));

-- name: GetPhoneRegistrationCandidate :one
-- Only picks a phone advisory lock. This result is NOT authentication.
SELECT phone FROM phone_registration_challenges
WHERE challenge_hash = $1;

-- name: LockActivePhoneRegistrationChallenge :one
SELECT id, phone, nonce, otp_mac
FROM phone_registration_challenges
WHERE challenge_hash = sqlc.arg(challenge_hash)
  AND phone = sqlc.arg(phone)
  AND consumed_at IS NULL AND expires_at > now() AND attempts < 5
FOR UPDATE;

-- name: CountPhoneRegistrationAttempt :exec
UPDATE phone_registration_challenges
SET attempts = attempts + 1
WHERE id = $1 AND consumed_at IS NULL AND attempts < 5;

-- name: ConsumePhoneRegistrationChallenge :exec
UPDATE phone_registration_challenges SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL;

-- name: CreatePrimaryVerifiedAccountPhone :exec
INSERT INTO account_phones (account_id, phone, verified_at, is_primary)
VALUES (sqlc.arg(account_id), sqlc.arg(phone), now(), true);
