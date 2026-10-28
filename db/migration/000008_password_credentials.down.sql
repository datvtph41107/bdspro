-- Reject a downgrade if Google-only accounts cannot supply a password.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM accounts a LEFT JOIN password_credentials p
          ON p.account_id = a.id
        WHERE p.account_id IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot downgrade passwordless accounts';
    END IF;
END $$;

ALTER TABLE accounts ADD COLUMN password_hash text;
UPDATE accounts a SET password_hash = p.password_hash
FROM password_credentials p WHERE p.account_id = a.id;
ALTER TABLE accounts ALTER COLUMN password_hash SET NOT NULL;
DROP TABLE password_credentials;
