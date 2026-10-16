-- name: CreateAccount :one
INSERT INTO accounts (person_id) VALUES ($1) RETURNING id;

-- name: CreatePrimaryAccountEmail :one
INSERT INTO account_emails (account_id, email, is_primary)
VALUES (sqlc.arg(account_id), sqlc.arg(email), true) RETURNING id;

-- name: CreatePasswordCredential :exec
INSERT INTO password_credentials (account_id, password_hash)
VALUES (sqlc.arg(account_id), sqlc.arg(password_hash));

-- name: GetLoginCredentials :one
SELECT a.id AS account_id, pc.password_hash, (ae.verified_at IS NOT NULL) AS email_verified
FROM accounts a
JOIN account_emails ae ON ae.account_id = a.id AND ae.is_primary
JOIN password_credentials pc ON pc.account_id = a.id
WHERE lower(btrim(ae.email)) = $1 AND a.closed_at IS NULL;

-- name: GetOpenAccountPersonID :one
SELECT person_id FROM accounts WHERE id = $1 AND closed_at IS NULL;

-- name: GetInvitationAccountPersonID :one
SELECT person_id FROM accounts WHERE id = $1;
