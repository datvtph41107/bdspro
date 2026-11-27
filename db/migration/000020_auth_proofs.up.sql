CREATE TABLE auth_proofs (
    digest bytea PRIMARY KEY CHECK (octet_length(digest)=32),
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('password.set','email.link','phone.link','google.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all','organization.owner.transfer')),
    method text NOT NULL CHECK (method IN ('password','verified_email','google')),
    email_snapshot text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    google_subject text NOT NULL DEFAULT '',
    CONSTRAINT auth_proofs_method_binding_check CHECK (
    (method='verified_email' AND action='password.set' AND email_snapshot<>'' AND google_subject='') OR
    (method='password' AND email_snapshot='' AND google_subject='') OR
    (method='google' AND email_snapshot='' AND google_subject<>'' AND action IN
        ('password.set','email.link','phone.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all'))
)
);
CREATE INDEX auth_proofs_account_idx ON auth_proofs(account_id);
CREATE INDEX auth_proofs_expiry_idx ON auth_proofs(expires_at);
