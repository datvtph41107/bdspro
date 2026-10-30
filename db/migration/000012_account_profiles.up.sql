-- A person has one editable profile. Account settings belong to authentication,
-- not to the person (who may have other business relationships).
CREATE TABLE person_profiles (
    person_id bigint PRIMARY KEY REFERENCES persons(party_id) ON DELETE CASCADE,
    display_name text NOT NULL DEFAULT '' CHECK (length(display_name) <= 120),
    first_name text NOT NULL DEFAULT '' CHECK (length(first_name) <= 120),
    last_name text NOT NULL DEFAULT '' CHECK (length(last_name) <= 120),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE account_settings (
    account_id bigint PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    locale text NOT NULL DEFAULT 'vi' CHECK (locale IN ('vi', 'en')),
    timezone text NOT NULL DEFAULT 'Asia/Ho_Chi_Minh' CHECK (timezone IN ('Asia/Ho_Chi_Minh', 'UTC')),
    currency text NOT NULL DEFAULT 'VND' CHECK (currency IN ('VND', 'USD')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Existing accounts get empty (not fabricated) names and conservative defaults.
-- Their created_at is not backdated to pretend we know when the profile existed.
INSERT INTO person_profiles (person_id)
SELECT party_id FROM persons;

INSERT INTO account_settings (account_id)
SELECT id FROM accounts;
