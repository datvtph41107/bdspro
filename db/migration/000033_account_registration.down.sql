DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM accounts WHERE username IS NOT NULL OR status <> 'active' OR terms_version IS NOT NULL OR marketing_opt_in)
       OR EXISTS (SELECT 1 FROM email_registration_challenges) THEN
        RAISE EXCEPTION 'cannot roll back IAM registration with identity or pending proof data';
    END IF;
END $$;
DROP TABLE email_registration_challenges;
DROP INDEX accounts_username_uq;
ALTER TABLE accounts DROP COLUMN username, DROP COLUMN status, DROP COLUMN terms_version,
    DROP COLUMN terms_accepted_at, DROP COLUMN marketing_opt_in;
