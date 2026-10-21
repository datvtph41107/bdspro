CREATE TABLE organizations (
  party_id bigint PRIMARY KEY REFERENCES parties(id),
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE organization_memberships (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations(party_id),
  person_id bigint NOT NULL REFERENCES persons(party_id),
  role text NOT NULL CHECK (role IN ('owner', 'member')),
  joined_at timestamptz NOT NULL DEFAULT now(),
  ended_at timestamptz
);

CREATE UNIQUE INDEX organization_memberships_one_open_uq
  ON organization_memberships (organization_id, person_id) WHERE ended_at IS NULL;

CREATE INDEX organization_memberships_person_idx
  ON organization_memberships (person_id);
