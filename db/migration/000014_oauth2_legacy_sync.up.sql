-- Transitional compatibility for deployments rolling from google_identities
-- to issuer-scoped oauth2_identities. Remove this bridge only together with
-- removal of the legacy code path. Do not fork account ownership.
CREATE UNIQUE INDEX oauth2_identities_account_issuer_uq
    ON oauth2_identities (account_id, issuer);

CREATE FUNCTION sync_legacy_identity_to_oauth2() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM oauth2_identities
         WHERE issuer = 'https://accounts.google.com'
           AND subject = OLD.google_sub AND account_id = OLD.account_id;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE'
       AND (OLD.google_sub IS DISTINCT FROM NEW.google_sub
            OR OLD.account_id IS DISTINCT FROM NEW.account_id) THEN
        DELETE FROM oauth2_identities
         WHERE issuer = 'https://accounts.google.com'
           AND subject = OLD.google_sub AND account_id = OLD.account_id;
    END IF;

    INSERT INTO oauth2_identities (issuer, subject, account_id, linked_at)
    VALUES ('https://accounts.google.com', NEW.google_sub, NEW.account_id, NEW.linked_at)
    ON CONFLICT (issuer, subject) DO UPDATE
       SET linked_at = EXCLUDED.linked_at
     WHERE oauth2_identities.account_id = EXCLUDED.account_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'external identity belongs to another account'
            USING ERRCODE = '23505';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER google_identity_to_oauth2
AFTER INSERT OR UPDATE OR DELETE ON google_identities
FOR EACH ROW EXECUTE FUNCTION sync_legacy_identity_to_oauth2();

CREATE FUNCTION sync_oauth2_identity_to_legacy() RETURNS trigger
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

-- Do not subscribe to UPDATE: conflict-update operations may be re-entrant.
CREATE TRIGGER oauth2_identity_to_legacy
AFTER INSERT OR DELETE ON oauth2_identities
FOR EACH ROW EXECUTE FUNCTION sync_oauth2_identity_to_legacy();
