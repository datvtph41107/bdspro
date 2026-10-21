-- name: CreateVerifyEmail :one
INSERT INTO verify_emails (account_id, email, secret_code)
VALUES ($1, $2, $3)
RETURNING id, account_id, email, secret_code, is_used, created_at, expired_at;

-- name: ConsumeVerifyEmail :one
UPDATE verify_emails SET is_used = TRUE
WHERE id = $1 AND secret_code = $2 AND is_used = FALSE AND expired_at > now()
RETURNING id, account_id, email, secret_code, is_used, created_at, expired_at;
