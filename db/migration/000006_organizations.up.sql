CREATE TABLE organizations (
  party_id bigint PRIMARY KEY REFERENCES parties(id),
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
