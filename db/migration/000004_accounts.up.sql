CREATE TABLE accounts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    person_id bigint NOT NULL UNIQUE REFERENCES persons(party_id),
    email text,
    created_at timestamptz NOT NULL DEFAULT now(),
    is_email_verified boolean NOT NULL DEFAULT false,
    password_reset_version bigint NOT NULL DEFAULT 0 CHECK (password_reset_version >= 0),
    username text,
    status text NOT NULL DEFAULT 'active',
    terms_version text,
    terms_accepted_at timestamptz,
    marketing_opt_in boolean NOT NULL DEFAULT false,
    CONSTRAINT accounts_username_check CHECK (username IS NULL OR (username ~ '^[a-z0-9_-]{3,30}$' AND username NOT IN ('admin','system','support'))),
    CONSTRAINT accounts_status_check CHECK (status IN ('active','suspended')),
    CONSTRAINT accounts_terms_check CHECK ((terms_version IS NULL) = (terms_accepted_at IS NULL) AND (terms_version IS NULL OR length(terms_version) BETWEEN 1 AND 64))
);

CREATE UNIQUE INDEX accounts_email_login_uq ON accounts (lower(btrim(email)));
CREATE UNIQUE INDEX accounts_username_uq ON accounts(username) WHERE username IS NOT NULL;
