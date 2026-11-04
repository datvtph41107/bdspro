-- Teams are subordinate to organizations. Organization membership remains
-- the authority for account access; Team membership never supersedes it.
CREATE TABLE teams (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations(party_id) ON DELETE CASCADE,
  name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
  description text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX teams_org_name_uq ON teams (organization_id, lower(btrim(name)));
CREATE INDEX teams_org_idx ON teams (organization_id, id);

CREATE TABLE team_members (
  team_id bigint NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  person_id bigint NOT NULL REFERENCES persons(party_id),
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (team_id, person_id)
);

CREATE INDEX team_members_person_idx ON team_members (person_id);

-- Store only a hash of the invitation proof, never its raw token.
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
  CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);

CREATE UNIQUE INDEX team_invites_open_uq
  ON team_invites (team_id, lower(btrim(invited_email)))
  WHERE accepted_at IS NULL AND revoked_at IS NULL;
CREATE INDEX team_invites_team_idx ON team_invites (team_id, created_at);
