CREATE TABLE google_authenticated_intents (
    browser_digest bytea PRIMARY KEY REFERENCES google_auth_flows(browser_digest) ON DELETE CASCADE,
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('link','reauth')),
    action text NOT NULL,
    CHECK ((purpose='link' AND action='google.link') OR
           (purpose='reauth' AND action IN ('password.set','email.link','phone.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all')))
);
CREATE INDEX google_authenticated_intents_account_idx ON google_authenticated_intents(account_id);
ALTER TABLE google_auth_flows ADD COLUMN auth_time bigint NOT NULL DEFAULT 0 CHECK (auth_time>=0);
ALTER TABLE recent_auth_proofs ADD COLUMN google_subject text NOT NULL DEFAULT '';
ALTER TABLE recent_auth_proofs DROP CONSTRAINT recent_auth_proofs_method_check;
ALTER TABLE recent_auth_proofs DROP CONSTRAINT recent_auth_proofs_check;
ALTER TABLE recent_auth_proofs ADD CONSTRAINT recent_auth_proofs_method_check CHECK (method IN ('password','verified_email','google'));
ALTER TABLE recent_auth_proofs ADD CONSTRAINT recent_auth_proofs_method_binding_check CHECK (
    (method='verified_email' AND action='password.set' AND email_snapshot<>'' AND google_subject='') OR
    (method='password' AND email_snapshot='' AND google_subject='') OR
    (method='google' AND email_snapshot='' AND google_subject<>'' AND action IN
        ('password.set','email.link','phone.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all'))
);
