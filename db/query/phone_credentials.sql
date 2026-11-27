-- name: GetPhoneAccountIDQuery :one
SELECT account_id FROM phone_credentials WHERE phone_e164 = $1;

-- name: BindPhoneCredentialQuery :exec
INSERT INTO phone_credentials (account_id, phone_e164, verified_at)
VALUES (sqlc.arg(account_id), sqlc.arg(phone_e164), now());

-- name: LockPhoneCredentialQuery :one
SELECT account_id FROM phone_credentials WHERE phone_e164 = $1 FOR SHARE;

-- name: GetPhoneForAccount :one
SELECT phone_e164 FROM phone_credentials WHERE account_id = $1;

-- name: RemoveAccountPhone :execrows
DELETE FROM phone_credentials WHERE account_id=$1;

-- name: SetVerifiedAccountPhone :exec
INSERT INTO phone_credentials(account_id,phone_e164,verified_at)
VALUES (sqlc.arg(account_id)::bigint,sqlc.arg(phone_e164)::text,clock_timestamp())
ON CONFLICT (account_id) DO UPDATE SET phone_e164=EXCLUDED.phone_e164,verified_at=EXCLUDED.verified_at;
