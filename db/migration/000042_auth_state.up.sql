BEGIN;
-- Hold source rows stable through copy and drop; deploy with old writers stopped.
LOCK TABLE email_registration_challenges, phone_continuation_challenges, google_auth_flows, google_authenticated_intents IN ACCESS EXCLUSIVE MODE;

-- Email and phone have the same browser-bound, single-use lifecycle.
-- The composite key preserves even coincident UUIDs/proof digests across kinds.
CREATE TABLE verifications (
    kind text NOT NULL CHECK (kind IN ('email', 'phone')),
    id uuid NOT NULL,
    destination text NOT NULL,
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest) = 32),
    code_mac bytea CHECK (octet_length(code_mac) = 32),
    proof_digest bytea CHECK (octet_length(proof_digest) = 32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, id),
    UNIQUE (kind, proof_digest),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL)),
    CHECK ((kind = 'email' AND length(destination) BETWEEN 3 AND 254 AND destination = lower(btrim(destination)))
        OR (kind = 'phone' AND destination ~ '^\+[1-9][0-9]{7,14}$'))
);
CREATE INDEX verifications_expiry_idx ON verifications(kind, expires_at);

INSERT INTO verifications
SELECT 'email', id, email, browser_digest, code_mac, proof_digest, expires_at, resend_after, failed_attempts, created_at
FROM email_registration_challenges;
INSERT INTO verifications
SELECT 'phone', id, phone_e164, browser_digest, code_mac, proof_digest, expires_at, resend_after, failed_attempts, created_at
FROM phone_continuation_challenges;

DROP TABLE email_registration_challenges;
DROP TABLE phone_continuation_challenges;

ALTER TABLE google_auth_flows RENAME TO google_flows;
ALTER TABLE google_flows
    ADD COLUMN account_id bigint REFERENCES accounts(id) ON DELETE CASCADE,
    ADD COLUMN session_id uuid REFERENCES sessions(id) ON DELETE CASCADE,
    ADD COLUMN purpose text NOT NULL DEFAULT 'login',
    ADD COLUMN action text NOT NULL DEFAULT '';

UPDATE google_flows f SET account_id = i.account_id, session_id = i.session_id,
    purpose = i.purpose, action = i.action
FROM google_authenticated_intents i WHERE i.browser_digest = f.browser_digest;

ALTER TABLE google_flows ADD CONSTRAINT google_flows_intent_check CHECK (
    (purpose = 'login' AND account_id IS NULL AND session_id IS NULL AND action = '')
    OR (account_id IS NOT NULL AND session_id IS NOT NULL AND (
        (purpose = 'link' AND action = 'google.link')
        OR (purpose = 'reauth' AND action IN ('password.set','email.link','phone.link','email.unlink','phone.unlink','google.unlink','password.unlink','sessions.revoke_all'))
    ))
);
CREATE INDEX google_flows_account_idx ON google_flows(account_id) WHERE account_id IS NOT NULL;
DROP TABLE google_authenticated_intents;

ALTER TABLE recent_auth_proofs RENAME TO auth_proofs;
ALTER TABLE credential_link_challenges RENAME TO contact_links;

-- Historical rows cannot be assigned an invented browser/session binding.
-- Retain them; no current application queries use these two tables.
COMMENT ON TABLE phone_otp_challenges IS 'Legacy storage; no current runtime reader/writer. Retained to preserve historical data.';
COMMENT ON TABLE phone_signup_challenges IS 'Legacy storage; no current runtime reader/writer. Retained to preserve historical data.';
COMMIT;
