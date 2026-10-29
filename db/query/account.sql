-- name: CreateAccount :one
INSERT INTO accounts (person_id, email)
VALUES ($1, $2)
RETURNING id, person_id, email, created_at, is_email_verified;

-- name: CreateVerifiedGoogleAccount :one
INSERT INTO accounts (person_id, email, is_email_verified)
VALUES ($1, $2, TRUE)
RETURNING id, person_id, email, created_at, is_email_verified;

-- name: ReadAccountByEmail :one
SELECT id, person_id, COALESCE(email, '')::text AS email, created_at, is_email_verified
FROM accounts WHERE lower(btrim(email)) = lower(btrim($1)) LIMIT 1;

-- name: ReadAccountByID :one
SELECT id, person_id, COALESCE(email, '')::text AS email, created_at, is_email_verified
FROM accounts WHERE id = $1 LIMIT 1;

-- name: MarkEmailVerified :one
UPDATE accounts SET is_email_verified = TRUE
WHERE id = sqlc.arg(id) AND lower(btrim(email)) = lower(btrim(sqlc.arg(email)))
RETURNING id, person_id, email, created_at, is_email_verified;
