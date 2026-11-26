-- name: CreateAccount :one
INSERT INTO accounts (person_id, email)
VALUES ($1, $2)
RETURNING id, person_id, email, created_at, is_email_verified, password_reset_version;

-- name: ReadAccountByEmail :one
SELECT id, person_id, COALESCE(email, '')::text AS email, created_at, is_email_verified
FROM accounts WHERE lower(btrim(email)) = lower(btrim($1)) LIMIT 1;

-- name: ReadAccountByID :one
SELECT id, person_id, COALESCE(email, '')::text AS email, created_at, is_email_verified
FROM accounts WHERE id = $1 LIMIT 1;

-- name: MarkEmailVerified :one
UPDATE accounts SET is_email_verified = TRUE
WHERE id = sqlc.arg(id) AND lower(btrim(email)) = lower(btrim(sqlc.arg(email)))
RETURNING id, person_id, email, created_at, is_email_verified, password_reset_version;

-- name: GetAccountIdentity :one
SELECT COALESCE(username,'')::text AS username, status, COALESCE(terms_version,'')::text AS terms_version, marketing_opt_in
FROM accounts WHERE id=$1;

-- name: ReadAccountIDByUsername :one
SELECT id FROM accounts WHERE username=sqlc.arg(username)::text;

-- name: SetRegistrationIdentity :execrows
UPDATE accounts
SET username=sqlc.arg(username)::text, terms_version=sqlc.arg(terms_version)::text,
    terms_accepted_at=clock_timestamp(), marketing_opt_in=sqlc.arg(marketing_opt_in)::boolean
WHERE id=sqlc.arg(account_id)::bigint AND status='active' AND username IS NULL AND terms_version IS NULL;

-- name: SetAccountUsername :execrows
UPDATE accounts SET username=sqlc.arg(username)::text
WHERE id=sqlc.arg(account_id)::bigint AND status='active';

-- name: SetAccountMarketing :execrows
UPDATE accounts SET marketing_opt_in=sqlc.arg(marketing_opt_in)::boolean
WHERE id=sqlc.arg(account_id)::bigint AND status='active';

-- name: RemoveAccountEmail :execrows
UPDATE accounts SET email=NULL,is_email_verified=false WHERE id=$1 AND email IS NOT NULL;
