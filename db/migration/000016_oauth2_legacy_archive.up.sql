-- Single active OAuth2 credential source. Keep the old association rows
-- archived (not deleted) for controlled deployment rollback and audit.
DROP TRIGGER oauth2_credential_to_legacy ON oauth2_credentials;
DROP TRIGGER google_credential_to_oauth2 ON google_identities;
DROP FUNCTION sync_oauth2_credential_to_legacy();
DROP FUNCTION sync_legacy_credential_to_oauth2();
ALTER TABLE google_identities RENAME TO oauth2_credentials_archive;
ALTER TABLE oauth2_credentials_archive RENAME COLUMN google_sub TO subject;
ALTER INDEX google_identities_pkey RENAME TO oauth2_credentials_archive_pkey;
ALTER INDEX google_identities_account_id_key RENAME TO oauth2_credentials_archive_account_id_key;
