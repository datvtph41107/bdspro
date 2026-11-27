-- name: GetOAuth2AccountID :one
SELECT account_id FROM oauth2_credentials
WHERE issuer = sqlc.arg(issuer) AND subject = sqlc.arg(subject);

-- name: LinkOAuth2Credential :one
INSERT INTO oauth2_credentials (issuer, subject, account_id)
VALUES (sqlc.arg(issuer), sqlc.arg(subject), sqlc.arg(account_id))
ON CONFLICT (issuer, subject) DO UPDATE
SET subject = EXCLUDED.subject
WHERE oauth2_credentials.account_id = EXCLUDED.account_id
RETURNING account_id;

-- name: ListOAuth2CredentialsForAccount :many
SELECT issuer, subject, account_id, linked_at
FROM oauth2_credentials WHERE account_id = $1
ORDER BY linked_at, issuer, subject;

-- name: LockGoogleIdentityOwner :one
SELECT account_id FROM oauth2_credentials WHERE issuer=sqlc.arg(issuer)::text AND subject=sqlc.arg(subject)::text FOR SHARE;

-- name: RemoveAccountGoogle :execrows
DELETE FROM oauth2_credentials WHERE account_id=$1 AND issuer IN ('https://accounts.google.com','accounts.google.com');
