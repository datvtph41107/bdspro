CREATE TABLE team_invites (
    id uuid PRIMARY KEY,
    team_id bigint NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    invited_email text NOT NULL,
    invited_by_account_id bigint NOT NULL REFERENCES accounts(id),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    recipient_account_id bigint REFERENCES accounts(id),
    recipient_membership_id bigint REFERENCES organization_members(id),
    CHECK (accepted_at IS NULL OR revoked_at IS NULL),
    CONSTRAINT team_invites_recipient_binding_check
        CHECK ((recipient_account_id IS NULL) = (recipient_membership_id IS NULL))
);

CREATE UNIQUE INDEX team_invites_open_uq
    ON team_invites (team_id, lower(btrim(invited_email)))
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

CREATE INDEX team_invites_team_idx ON team_invites (team_id, created_at);
