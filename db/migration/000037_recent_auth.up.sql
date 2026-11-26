CREATE TABLE recent_auth_proofs (
    digest bytea PRIMARY KEY CHECK (octet_length(digest)=32),
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('password.set','email.link','phone.link','google.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all','organization.owner.transfer')),
    method text NOT NULL CHECK (method IN ('password','verified_email')),
    email_snapshot text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    CHECK ((method='verified_email' AND action='password.set' AND email_snapshot<>'') OR (method='password' AND email_snapshot=''))
);
CREATE INDEX recent_auth_proofs_account_idx ON recent_auth_proofs(account_id);
CREATE INDEX recent_auth_proofs_expiry_idx ON recent_auth_proofs(expires_at);
