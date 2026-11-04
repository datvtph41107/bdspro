-- Provider-independent OIDC identity storage. Preserve existing Google links while
-- clients migrate; the old table is intentionally not removed in this release.
CREATE TABLE oauth2_identities (
    issuer text NOT NULL CHECK (length(issuer) > 0),
    subject text NOT NULL CHECK (length(subject) > 0),
    account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    linked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (issuer, subject)
);
CREATE INDEX oauth2_identities_account_idx ON oauth2_identities(account_id);

INSERT INTO oauth2_identities (issuer, subject, account_id, linked_at)
SELECT 'https://accounts.google.com', google_sub, account_id, linked_at
FROM google_identities
ON CONFLICT (issuer, subject) DO NOTHING;
