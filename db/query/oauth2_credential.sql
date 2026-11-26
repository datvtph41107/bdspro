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

-- name: CreateGoogleFlow :exec
INSERT INTO google_auth_flows(browser_digest,state_digest,nonce,verifier,phase,remember,expires_at)
VALUES (sqlc.arg(browser_digest)::bytea,sqlc.arg(state_digest)::bytea,sqlc.arg(nonce)::text,sqlc.arg(verifier)::text,'pending',sqlc.arg(remember)::boolean,clock_timestamp()+interval '10 minutes');

-- name: LockPendingGoogleFlow :one
SELECT nonce,verifier, EXISTS (SELECT 1 FROM google_authenticated_intents i WHERE i.browser_digest=google_auth_flows.browser_digest AND i.purpose='reauth') AS requires_recent_auth FROM google_auth_flows
WHERE google_auth_flows.browser_digest=sqlc.arg(browser_digest)::bytea AND state_digest=sqlc.arg(state_digest)::bytea
AND phase='pending' AND expires_at>clock_timestamp() FOR UPDATE;

-- name: ClaimGoogleFlow :execrows
UPDATE google_auth_flows SET phase='exchanging',state_digest=NULL,nonce='',verifier='',expires_at=clock_timestamp()+interval '2 minutes'
WHERE browser_digest=$1 AND phase='pending' AND expires_at>clock_timestamp();

-- name: FinishGoogleFlow :execrows
UPDATE google_auth_flows SET phase='ready',subject=sqlc.arg(subject)::text,email=sqlc.arg(email)::text,
    authoritative_email=sqlc.arg(authoritative_email)::boolean,auth_time=sqlc.arg(auth_time)::bigint,expires_at=clock_timestamp()+interval '5 minutes'
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND phase='exchanging' AND expires_at>clock_timestamp();

-- name: LockGoogleResult :one
SELECT subject,email,authoritative_email,remember FROM google_auth_flows
WHERE google_auth_flows.browser_digest=$1 AND phase='ready' AND expires_at>clock_timestamp()
AND NOT EXISTS (SELECT 1 FROM google_authenticated_intents i WHERE i.browser_digest=google_auth_flows.browser_digest) FOR UPDATE;

-- name: IssueGoogleRegistrationProof :execrows
UPDATE google_auth_flows SET phase='registration',proof_digest=sqlc.arg(proof_digest)::bytea,expires_at=clock_timestamp()+interval '15 minutes'
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND phase='ready' AND expires_at>clock_timestamp();

-- name: LockGoogleRegistration :one
SELECT subject,email,authoritative_email,remember FROM google_auth_flows
WHERE google_auth_flows.browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest=sqlc.arg(proof_digest)::bytea
AND phase='registration' AND expires_at>clock_timestamp()
AND NOT EXISTS (SELECT 1 FROM google_authenticated_intents i WHERE i.browser_digest=google_auth_flows.browser_digest) FOR UPDATE;

-- name: DeleteGoogleFlow :exec
DELETE FROM google_auth_flows WHERE browser_digest=$1;

-- name: DeleteExpiredGoogleFlows :execrows
DELETE FROM google_auth_flows WHERE browser_digest IN (
    SELECT browser_digest FROM google_auth_flows WHERE expires_at<clock_timestamp()
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);

-- name: LockGoogleIdentityOwner :one
SELECT account_id FROM oauth2_credentials WHERE issuer=sqlc.arg(issuer)::text AND subject=sqlc.arg(subject)::text FOR SHARE;

-- name: RemoveAccountGoogle :execrows
DELETE FROM oauth2_credentials WHERE account_id=$1 AND issuer IN ('https://accounts.google.com','accounts.google.com');

-- name: CreateGoogleAuthenticatedIntent :exec
INSERT INTO google_authenticated_intents(browser_digest,account_id,session_id,purpose,action)
VALUES (sqlc.arg(browser_digest)::bytea,sqlc.arg(account_id)::bigint,sqlc.arg(session_id)::uuid,sqlc.arg(purpose)::text,sqlc.arg(action)::text);

-- name: LockGoogleAuthenticatedResult :one
SELECT f.subject,f.auth_time,i.action FROM google_auth_flows f
JOIN google_authenticated_intents i ON i.browser_digest=f.browser_digest
WHERE f.browser_digest=sqlc.arg(browser_digest)::bytea AND i.account_id=sqlc.arg(account_id)::bigint
AND i.session_id=sqlc.arg(session_id)::uuid AND i.purpose=sqlc.arg(purpose)::text
AND f.phase='ready' AND f.expires_at>clock_timestamp() FOR UPDATE OF f;

-- name: DeleteAccountGoogleIntents :exec
DELETE FROM google_auth_flows WHERE google_auth_flows.browser_digest IN
    (SELECT i.browser_digest FROM google_authenticated_intents i WHERE i.account_id=$1);
