-- name: LinkGoogleIdentityQuery :one
INSERT INTO google_identities (account_id, google_sub)
VALUES (sqlc.arg(account_id), sqlc.arg(google_sub))
ON CONFLICT (google_sub) DO UPDATE
SET google_sub = EXCLUDED.google_sub
WHERE google_identities.account_id = EXCLUDED.account_id
RETURNING account_id;

-- name: GetGoogleAccountIDQuery :one
SELECT account_id
FROM google_identities
WHERE google_sub = $1;
