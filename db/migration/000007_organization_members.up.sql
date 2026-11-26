CREATE TABLE organization_members (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations(party_id),
  person_id bigint NOT NULL REFERENCES persons(party_id),
  role text NOT NULL CHECK (role IN ('owner', 'member')),
  joined_at timestamptz NOT NULL DEFAULT now(),
  ended_at timestamptz
);

CREATE UNIQUE INDEX organization_members_one_open_uq
  ON organization_members (organization_id, person_id) WHERE ended_at IS NULL;

CREATE INDEX organization_members_person_idx
  ON organization_members (person_id);
