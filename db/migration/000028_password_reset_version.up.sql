ALTER TABLE accounts
    ADD COLUMN password_reset_version bigint NOT NULL DEFAULT 0
    CHECK (password_reset_version >= 0);
