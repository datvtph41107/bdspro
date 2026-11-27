BEGIN;
-- Hold source rows stable through copy and drop; deploy with old writers stopped.
LOCK TABLE verifications, google_flows IN ACCESS EXCLUSIVE MODE;
ALTER TABLE auth_proofs RENAME TO recent_auth_proofs;
ALTER TABLE contact_links RENAME TO credential_link_challenges;
ALTER TABLE google_flows RENAME TO google_auth_flows;
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

INSERT INTO google_authenticated_intents(browser_digest,account_id,session_id,purpose,action)
SELECT browser_digest,account_id,session_id,purpose,action FROM google_auth_flows WHERE purpose <> 'login';
ALTER TABLE google_auth_flows DROP CONSTRAINT google_flows_intent_check;
DROP INDEX google_flows_account_idx;
ALTER TABLE google_auth_flows DROP COLUMN account_id, DROP COLUMN session_id, DROP COLUMN purpose, DROP COLUMN action;
CREATE TABLE email_registration_challenges (
    id uuid PRIMARY KEY,
    email text NOT NULL CHECK (length(email) BETWEEN 3 AND 254 AND email=lower(btrim(email))),
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest)=32),
    code_mac bytea CHECK (octet_length(code_mac)=32),
    proof_digest bytea UNIQUE CHECK (octet_length(proof_digest)=32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL))
);
CREATE INDEX email_registration_expiry_idx ON email_registration_challenges(expires_at);
CREATE TABLE phone_continuation_challenges (
    id uuid PRIMARY KEY,
    phone_e164 text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
    browser_digest bytea NOT NULL CHECK (octet_length(browser_digest)=32),
    code_mac bytea CHECK (octet_length(code_mac)=32),
    proof_digest bytea UNIQUE CHECK (octet_length(proof_digest)=32),
    expires_at timestamptz NOT NULL,
    resend_after timestamptz NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((code_mac IS NULL) <> (proof_digest IS NULL))
);
CREATE INDEX phone_continuation_expiry_idx ON phone_continuation_challenges(expires_at);

INSERT INTO email_registration_challenges
SELECT id,destination,browser_digest,code_mac,proof_digest,expires_at,resend_after,failed_attempts,created_at
FROM verifications WHERE kind='email';
INSERT INTO phone_continuation_challenges
SELECT id,destination,browser_digest,code_mac,proof_digest,expires_at,resend_after,failed_attempts,created_at
FROM verifications WHERE kind='phone';
DROP TABLE verifications;
COMMENT ON TABLE phone_otp_challenges IS NULL;
COMMENT ON TABLE phone_signup_challenges IS NULL;
COMMIT;
