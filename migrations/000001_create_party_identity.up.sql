-- CORE-01 / C01 baseline. Intended for a new EMPTY PostgreSQL database.
-- Party kind is durable: a Person CANNOT become an Organization (or vice versa).
-- Creating an Organization row does NOT certify legal incorporation/representation.
BEGIN;

CREATE TABLE parties (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

CREATE TABLE persons (
    party_id bigint PRIMARY KEY
        REFERENCES parties(id) ON DELETE RESTRICT
);

CREATE TABLE organizations (
    party_id bigint PRIMARY KEY
        REFERENCES parties(id) ON DELETE RESTRICT,
    name text NOT NULL
        CONSTRAINT organizations_name_nonblank CHECK (btrim(name) <> '')
);

-- Prevent two writers from making independent subtype assertions for one Party.
-- FOR UPDATE serializes the subtype write against other writers of this Party.
CREATE FUNCTION party_lock_for_subtype_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM 1 FROM parties WHERE id = NEW.party_id FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'party % does not exist', NEW.party_id
            USING ERRCODE = '23503';
    END IF;

    IF TG_TABLE_NAME = 'persons' THEN
        IF EXISTS (SELECT 1 FROM organizations WHERE party_id = NEW.party_id) THEN
            RAISE EXCEPTION 'party % already has Organization subtype', NEW.party_id
                USING ERRCODE = '23514';
        END IF;
    ELSIF TG_TABLE_NAME = 'organizations' THEN
        IF EXISTS (SELECT 1 FROM persons WHERE party_id = NEW.party_id) THEN
            RAISE EXCEPTION 'party % already has Person subtype', NEW.party_id
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER persons_lock_on_insert
BEFORE INSERT ON persons
FOR EACH ROW EXECUTE FUNCTION party_lock_for_subtype_insert();

CREATE TRIGGER organizations_lock_on_insert
BEFORE INSERT ON organizations
FOR EACH ROW EXECUTE FUNCTION party_lock_for_subtype_insert();

-- The subtype relationship is identity, NOT a changeable organization role.
-- A human later creating an enterprise must create another Party ID.
CREATE FUNCTION party_prevent_subtype_change()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'party subtype is immutable: % on % is not allowed',
        TG_OP, TG_TABLE_NAME
        USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER persons_immutable_identity
BEFORE UPDATE OF party_id OR DELETE ON persons
FOR EACH ROW EXECUTE FUNCTION party_prevent_subtype_change();

CREATE TRIGGER organizations_immutable_identity
BEFORE UPDATE OF party_id OR DELETE ON organizations
FOR EACH ROW EXECUTE FUNCTION party_prevent_subtype_change();

-- FK only proves subtype -> Party existence. A deferred constraint trigger
-- proves the reverse TOTALITY at commit: 1 Party = exactly 1 subtype.
CREATE FUNCTION party_require_exactly_one_subtype()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    total integer;
BEGIN
    SELECT
        (EXISTS (SELECT 1 FROM persons WHERE party_id = NEW.id))::integer
      + (EXISTS (SELECT 1 FROM organizations WHERE party_id = NEW.id))::integer
    INTO total;

    IF total <> 1 THEN
        RAISE EXCEPTION 'party % must have exactly one subtype; found %',
            NEW.id, total
            USING ERRCODE = '23514',
                  CONSTRAINT = 'parties_exactly_one_subtype';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER parties_exactly_one_subtype
AFTER INSERT ON parties
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION party_require_exactly_one_subtype();

COMMIT;
