-- name: CreateGoogleFlow :exec
INSERT INTO google_flows(browser_digest,state_digest,nonce,verifier,phase,remember,expires_at)
VALUES (sqlc.arg(browser_digest)::bytea,sqlc.arg(state_digest)::bytea,sqlc.arg(nonce)::text,sqlc.arg(verifier)::text,'pending',sqlc.arg(remember)::boolean,clock_timestamp()+interval '10 minutes');

-- name: LockPendingGoogleFlow :one
SELECT nonce,verifier, (purpose='reauth') AS requires_recent_auth FROM google_flows
WHERE google_flows.browser_digest=sqlc.arg(browser_digest)::bytea AND state_digest=sqlc.arg(state_digest)::bytea
AND phase='pending' AND expires_at>clock_timestamp() FOR UPDATE;

-- name: ClaimGoogleFlow :execrows
UPDATE google_flows SET phase='exchanging',state_digest=NULL,nonce='',verifier='',expires_at=clock_timestamp()+interval '2 minutes'
WHERE browser_digest=$1 AND phase='pending' AND expires_at>clock_timestamp();

-- name: FinishGoogleFlow :execrows
UPDATE google_flows SET phase='ready',subject=sqlc.arg(subject)::text,email=sqlc.arg(email)::text,
    authoritative_email=sqlc.arg(authoritative_email)::boolean,auth_time=sqlc.arg(auth_time)::bigint,expires_at=clock_timestamp()+interval '5 minutes'
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND phase='exchanging' AND expires_at>clock_timestamp();

-- name: LockGoogleResult :one
SELECT subject,email,authoritative_email,remember FROM google_flows
WHERE google_flows.browser_digest=$1 AND phase='ready' AND expires_at>clock_timestamp()
AND purpose='login' FOR UPDATE;

-- name: IssueGoogleRegistrationProof :execrows
UPDATE google_flows SET phase='registration',proof_digest=sqlc.arg(proof_digest)::bytea,expires_at=clock_timestamp()+(sqlc.arg(ttl_minutes)::integer * interval '1 minute')
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND phase='ready' AND expires_at>clock_timestamp();

-- name: LockGoogleRegistration :one
SELECT subject,email,authoritative_email,remember FROM google_flows
WHERE google_flows.browser_digest=sqlc.arg(browser_digest)::bytea AND proof_digest=sqlc.arg(proof_digest)::bytea
AND phase='registration' AND expires_at>clock_timestamp()
AND purpose='login' FOR UPDATE;

-- name: DeleteGoogleFlow :exec
DELETE FROM google_flows WHERE browser_digest=$1;

-- name: DeleteExpiredGoogleFlows :execrows
DELETE FROM google_flows WHERE browser_digest IN (
    SELECT browser_digest FROM google_flows WHERE expires_at<clock_timestamp()
    ORDER BY expires_at LIMIT sqlc.arg(take)::integer FOR UPDATE SKIP LOCKED
);

-- name: BindGoogleFlow :execrows
UPDATE google_flows SET account_id=sqlc.arg(account_id)::bigint,
    session_id=sqlc.arg(session_id)::uuid,purpose=sqlc.arg(purpose)::text,action=sqlc.arg(action)::text
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND purpose='login' AND phase='pending';

-- name: LockGoogleAuthenticatedResult :one
SELECT subject,auth_time,action FROM google_flows
WHERE browser_digest=sqlc.arg(browser_digest)::bytea AND account_id=sqlc.arg(account_id)::bigint
AND session_id=sqlc.arg(session_id)::uuid AND purpose=sqlc.arg(purpose)::text
AND phase='ready' AND expires_at>clock_timestamp() FOR UPDATE;

-- name: DeleteAccountGoogleFlows :exec
DELETE FROM google_flows WHERE account_id=sqlc.arg(account_id)::bigint;
