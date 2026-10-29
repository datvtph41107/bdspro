-- Phone-only accounts can exist after this migration. Restoring NOT NULL
-- requires explicit conversion/migration of those records and is not automatic.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM accounts WHERE email IS NULL) THEN
   RAISE EXCEPTION 'cannot restore accounts.email NOT NULL while phone-only accounts exist';
 END IF;
END $$;
ALTER TABLE accounts ALTER COLUMN email SET NOT NULL;
DROP TABLE password_reset_tokens;
DROP TABLE phone_signup_challenges;
