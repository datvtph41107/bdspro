-- name: CreateAccount :one
INSERT INTO accounts (person_id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, person_id, email, password_hash, created_at;

-- name: GetAccountByEmail :one
SELECT id, person_id, email, password_hash, created_at
FROM accounts WHERE lower(btrim(email)) = lower(btrim($1)) LIMIT 1;
