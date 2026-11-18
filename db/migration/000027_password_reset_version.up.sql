-- A password recovery invalidates every previously requested reset job.
-- Increment under the same accounts FOR UPDATE lock used by recovery.
ALTER TABLE accounts
    ADD COLUMN password_reset_version bigint NOT NULL DEFAULT 0
    CHECK (password_reset_version >= 0);
