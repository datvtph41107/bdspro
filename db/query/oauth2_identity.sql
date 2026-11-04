-- name: GetOAuth2AccountID :one
SELECT account_id FROM oauth2_identities
WHERE issuer = sqlc.arg(issuer) AND subject = sqlc.arg(subject);

-- name: LinkOAuth2Identity :one
INSERT INTO oauth2_identities (issuer, subject, account_id)
VALUES (sqlc.arg(issuer), sqlc.arg(subject), sqlc.arg(account_id))
ON CONFLICT (issuer, subject) DO UPDATE
SET subject = EXCLUDED.subject
WHERE oauth2_identities.account_id = EXCLUDED.account_id
RETURNING account_id;

-- name: ListOAuth2IdentitiesForAccount :many
SELECT issuer, subject, account_id, linked_at
FROM oauth2_identities WHERE account_id = $1
ORDER BY linked_at, issuer, subject;
