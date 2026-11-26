-- name: CreatePasswordCredential :exec
INSERT INTO password_credentials (account_id, password_hash)
VALUES ($1, $2);

-- name: GetPasswordHash :one
SELECT password_hash
FROM password_credentials WHERE account_id = $1;

-- name: RemoveAccountPassword :execrows
DELETE FROM password_credentials WHERE account_id=$1;
