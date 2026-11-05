CREATE TABLE team_members (
  team_id bigint NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  person_id bigint NOT NULL REFERENCES persons(party_id),
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (team_id, person_id)
);

CREATE INDEX team_members_person_idx ON team_members (person_id);
