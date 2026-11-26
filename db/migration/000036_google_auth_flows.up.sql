CREATE TABLE google_auth_flows (
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
    CHECK ((phase='pending') = (state_digest IS NOT NULL)),
    CHECK ((phase='registration') = (proof_digest IS NOT NULL)),
    CHECK ((phase IN ('ready','registration')) = (subject <> '')),
    CHECK ((phase='pending' AND length(nonce)=43 AND length(verifier)=43) OR
           (phase<>'pending' AND nonce='' AND verifier=''))
);
CREATE INDEX google_auth_flows_expiry_idx ON google_auth_flows(expires_at);
