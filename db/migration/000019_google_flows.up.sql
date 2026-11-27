CREATE TABLE google_flows (
    browser_digest bytea PRIMARY KEY CHECK (octet_length(browser_digest)=32),
    state_digest bytea UNIQUE CHECK (octet_length(state_digest)=32),
    nonce text NOT NULL DEFAULT '',
    verifier text NOT NULL DEFAULT '',
    phase text NOT NULL CHECK (phase IN ('pending','exchanging','ready','registration')),
    subject text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    authoritative_email boolean NOT NULL DEFAULT false,
    proof_digest bytea UNIQUE CHECK (octet_length(proof_digest)=32),
    remember boolean NOT NULL,
    expires_at timestamptz NOT NULL,
    auth_time bigint NOT NULL DEFAULT 0 CHECK (auth_time>=0),
    account_id bigint REFERENCES accounts(id) ON DELETE CASCADE,
    session_id uuid REFERENCES sessions(id) ON DELETE CASCADE,
    purpose text NOT NULL DEFAULT 'login',
    action text NOT NULL DEFAULT '',
    CHECK ((phase='pending') = (state_digest IS NOT NULL)),
    CHECK ((phase='registration') = (proof_digest IS NOT NULL)),
    CHECK ((phase IN ('ready','registration')) = (subject <> '')),
    CHECK ((phase='pending' AND length(nonce)=43 AND length(verifier)=43) OR
           (phase<>'pending' AND nonce='' AND verifier='')),
    CONSTRAINT google_flows_intent_check CHECK (
    (purpose = 'login' AND account_id IS NULL AND session_id IS NULL AND action = '')
    OR (account_id IS NOT NULL AND session_id IS NOT NULL AND (
        (purpose = 'link' AND action = 'google.link')
        OR (purpose = 'reauth' AND action IN ('password.set','email.link','phone.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all'))
    ))
)
);
CREATE INDEX google_flows_expiry_idx ON google_flows(expires_at);

CREATE INDEX google_flows_account_idx ON google_flows(account_id) WHERE account_id IS NOT NULL;
