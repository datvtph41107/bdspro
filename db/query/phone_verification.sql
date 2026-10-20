-- name: UpsertAccountPhone :one
INSERT INTO account_phones(account_id, phone, is_primary)
VALUES (sqlc.arg(account_id), sqlc.arg(phone), false)
ON CONFLICT (account_id, phone) DO UPDATE SET phone = EXCLUDED.phone
RETURNING id;

-- name: LockOTPPhoneDestination :exec
-- Account row is locked first; this transaction-scoped advisory lock serializes
-- issuance to one phone number across distinct accounts.
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(phone)::text, 737681));

-- name: CanIssuePhoneChallenge :one
SELECT NOT EXISTS (
    SELECT 1 FROM phone_verification_challenges c
    WHERE c.account_phone_id = $1
      AND c.created_at > now() - interval '60 seconds'
) AND (
    SELECT count(*) FROM phone_verification_challenges c
    WHERE c.account_phone_id = $1
      AND c.created_at > now() - interval '24 hours'
) < 5 AND (
    SELECT count(*) FROM phone_verification_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.account_id = (
        SELECT account_id FROM account_phones WHERE id = $1
    ) AND recent.created_at > now() - interval '24 hours'
) < 10 AND NOT EXISTS (
    SELECT 1 FROM phone_verification_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.phone = (SELECT phone FROM account_phones WHERE id = $1)
      AND recent.created_at > now() - interval '60 seconds'
) AND (
    SELECT count(*) FROM phone_verification_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.phone = (SELECT phone FROM account_phones WHERE id = $1)
      AND recent.created_at > now() - interval '24 hours'
) < 5 AND NOT EXISTS (
    SELECT 1 FROM phone_login_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.phone = (SELECT phone FROM account_phones WHERE id = $1)
      AND recent.created_at > now() - interval '60 seconds'
) AND (
    SELECT count(*) FROM phone_login_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.phone = (SELECT phone FROM account_phones WHERE id = $1)
      AND recent.created_at > now() - interval '24 hours'
) + (
    SELECT count(*) FROM phone_verification_challenges recent
    JOIN account_phones ap ON ap.id = recent.account_phone_id
    WHERE ap.phone = (SELECT phone FROM account_phones WHERE id = $1)
      AND recent.created_at > now() - interval '24 hours'
) < 5 AS allowed;

-- name: InvalidatePriorPhoneChallenges :exec
UPDATE phone_verification_challenges SET consumed_at = now()
WHERE account_phone_id = $1 AND consumed_at IS NULL;

-- name: CreatePhoneChallenge :exec
INSERT INTO phone_verification_challenges(account_phone_id, nonce, otp_mac)
VALUES (sqlc.arg(account_phone_id), sqlc.arg(nonce), sqlc.arg(otp_mac));

-- name: GetOwnedAccountPhone :one
SELECT id FROM account_phones
WHERE account_id = sqlc.arg(account_id) AND phone = sqlc.arg(phone);

-- name: LockLatestPhoneChallenge :one
SELECT id, nonce, otp_mac
FROM phone_verification_challenges
WHERE account_phone_id = $1 AND consumed_at IS NULL
  AND expires_at > now() AND attempts < 5
ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE;

-- name: CountPhoneChallengeAttempt :exec
UPDATE phone_verification_challenges
SET attempts = attempts + 1
WHERE id = $1 AND consumed_at IS NULL AND attempts < 5;

-- name: ConsumePhoneChallenge :exec
UPDATE phone_verification_challenges SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL;

-- name: MarkAccountPhoneVerified :exec
UPDATE account_phones SET verified_at = COALESCE(verified_at, now())
WHERE id = $1;
