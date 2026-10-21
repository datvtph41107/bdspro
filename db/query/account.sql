-- name: CreateAccount :one
INSERT INTO accounts (person_id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, person_id, email, password_hash, created_at, is_email_verified;

-- name: GetAccountByEmail :one
SELECT id, person_id, email, password_hash, created_at, is_email_verified
FROM accounts WHERE lower(btrim(email)) = lower(btrim($1)) LIMIT 1;

-- name: GetAccountByID :one
SELECT id, person_id, email, password_hash, created_at, is_email_verified
FROM accounts WHERE id = $1 LIMIT 1;

-- name: MarkEmailVerified :one
UPDATE accounts SET is_email_verified = TRUE
WHERE id = $1 AND lower(btrim(email)) = lower(btrim($2))
RETURNING id, person_id, email, password_hash, created_at, is_email_verified;
