CREATE TABLE person_profiles (
    person_id bigint PRIMARY KEY REFERENCES persons(party_id) ON DELETE CASCADE,
    display_name text NOT NULL DEFAULT '' CHECK (length(display_name) <= 120),
    first_name text NOT NULL DEFAULT '' CHECK (length(first_name) <= 120),
    last_name text NOT NULL DEFAULT '' CHECK (length(last_name) <= 120),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
