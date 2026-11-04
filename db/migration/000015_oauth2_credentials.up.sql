-- Preserve every OAuth2 link and keep migration history immutable.
ALTER TABLE oauth2_identities RENAME TO oauth2_credentials;
ALTER INDEX oauth2_identities_pkey RENAME TO oauth2_credentials_pkey;
ALTER INDEX oauth2_identities_account_idx RENAME TO oauth2_credentials_account_idx;
ALTER INDEX oauth2_identities_account_issuer_uq RENAME TO oauth2_credentials_account_issuer_uq;

-- Keep legacy replication working during the rolling upgrade.
ALTER FUNCTION sync_legacy_identity_to_oauth2() RENAME TO sync_legacy_credential_to_oauth2;
ALTER FUNCTION sync_oauth2_identity_to_legacy() RENAME TO sync_oauth2_credential_to_legacy;
ALTER TRIGGER google_identity_to_oauth2 ON google_identities RENAME TO google_credential_to_oauth2;
ALTER TRIGGER oauth2_identity_to_legacy ON oauth2_credentials RENAME TO oauth2_credential_to_legacy;

CREATE OR REPLACE FUNCTION sync_legacy_credential_to_oauth2() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM oauth2_credentials
         WHERE issuer = 'https://accounts.google.com'
           AND subject = OLD.google_sub AND account_id = OLD.account_id;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE'
       AND (OLD.google_sub IS DISTINCT FROM NEW.google_sub
            OR OLD.account_id IS DISTINCT FROM NEW.account_id) THEN
        DELETE FROM oauth2_credentials
         WHERE issuer = 'https://accounts.google.com'
           AND subject = OLD.google_sub AND account_id = OLD.account_id;
    END IF;

    INSERT INTO oauth2_credentials (issuer, subject, account_id, linked_at)
    VALUES ('https://accounts.google.com', NEW.google_sub, NEW.account_id, NEW.linked_at)
    ON CONFLICT (issuer, subject) DO UPDATE
       SET linked_at = EXCLUDED.linked_at
     WHERE oauth2_credentials.account_id = EXCLUDED.account_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'external identity belongs to another account'
            USING ERRCODE = '23505';
    END IF;
    RETURN NEW;
END;
$$;


CREATE OR REPLACE FUNCTION sync_oauth2_credential_to_legacy() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD IS NOT NULL AND TG_OP = 'DELETE' THEN
        IF OLD.issuer = 'https://accounts.google.com' THEN
            DELETE FROM google_identities
             WHERE google_sub = OLD.subject AND account_id = OLD.account_id;
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.issuer <> 'https://accounts.google.com' THEN
        RETURN NEW;
    END IF;
    INSERT INTO google_identities (google_sub, account_id, linked_at)
    VALUES (NEW.subject, NEW.account_id, NEW.linked_at)
    ON CONFLICT (google_sub) DO UPDATE
       SET linked_at = EXCLUDED.linked_at
     WHERE google_identities.account_id = EXCLUDED.account_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'legacy identity belongs to another account'
            USING ERRCODE = '23505';
    END IF;
    RETURN NEW;
END;
$$;

