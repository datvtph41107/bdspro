DROP TRIGGER oauth2_identity_to_legacy ON oauth2_identities;
DROP FUNCTION sync_oauth2_identity_to_legacy();
DROP TRIGGER google_identity_to_oauth2 ON google_identities;
DROP FUNCTION sync_legacy_identity_to_oauth2();
DROP INDEX oauth2_identities_account_issuer_uq;
