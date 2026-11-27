-- name: LockVerifyEmailAccountQuery :one
SELECT is_email_verified FROM accounts WHERE id = $1 FOR UPDATE;

-- name: FindReusableVerifyEmailQuery :one
SELECT id, account_id, email, secret_code, is_used, created_at, expired_at
FROM verify_emails
WHERE account_id = sqlc.arg(account_id)
  AND email = sqlc.arg(email)
  AND is_used = FALSE
  AND expired_at > now()
ORDER BY id DESC
LIMIT 1;
