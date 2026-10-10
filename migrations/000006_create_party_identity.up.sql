-- CORE-01 / C01: each committed Party has exactly one subtype.
-- NOT a claim that an Organization's legal existence/representative authority is verified.
-- Intentionally separate from historical Listing migrations 000001..000005.
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

-- All subtype mutations acquire a row-level mutex on their parent Party.
-- FK KEY SHARE locks alone would not serialize Person/Organization inserts.
-- A committed subtype cannot be silently reassigned to a different Party.
CREATE FUNCTION party_lock_for_subtype_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    target_party_id bigint;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.party_id IS DISTINCT FROM OLD.party_id THEN
            RAISE EXCEPTION 'a subtype cannot move to another party'
                USING ERRCODE = '23514';
        END IF;
        target_party_id := NEW.party_id;
    ELSIF TG_OP = 'DELETE' THEN
        target_party_id := OLD.party_id;
    ELSE
        target_party_id := NEW.party_id;
    END IF;

    PERFORM 1 FROM parties WHERE id = target_party_id FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'party % does not exist', target_party_id
            USING ERRCODE = '23503';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER persons_lock_party
BEFORE INSERT OR DELETE OR UPDATE OF party_id ON persons
FOR EACH ROW EXECUTE FUNCTION party_lock_for_subtype_write();

CREATE TRIGGER organizations_lock_party
BEFORE INSERT OR DELETE OR UPDATE OF party_id ON organizations
FOR EACH ROW EXECUTE FUNCTION party_lock_for_subtype_write();

-- Constraint triggers are deferred to transaction end, permitting
-- INSERT parties + INSERT exactly one subtype in a single transaction.
-- The check ignores a Party removed in the same transaction. DELETE of
-- an existing Party still requires removing the subtype first (RESTRICT).
CREATE FUNCTION party_assert_exactly_one_subtype()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    target_party_id bigint;
    count_subtypes integer;
BEGIN
    IF TG_TABLE_NAME = 'parties' THEN
        target_party_id := NEW.id;
    ELSIF TG_OP = 'DELETE' THEN
        target_party_id := OLD.party_id;
    ELSE
        target_party_id := NEW.party_id;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM parties WHERE id = target_party_id) THEN
        RETURN NULL;
    END IF;

    SELECT (
        (EXISTS (SELECT 1 FROM persons WHERE party_id = target_party_id))::integer
        +
        (EXISTS (SELECT 1 FROM organizations WHERE party_id = target_party_id))::integer
    ) INTO count_subtypes;

    IF count_subtypes <> 1 THEN
        RAISE EXCEPTION 'party % must have exactly one subtype; found %',
            target_party_id, count_subtypes
            USING ERRCODE = '23514',
                  CONSTRAINT = 'parties_exactly_one_subtype';
    END IF;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER parties_exactly_one_subtype
AFTER INSERT ON parties
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION party_assert_exactly_one_subtype();

CREATE CONSTRAINT TRIGGER persons_exactly_one_subtype
AFTER INSERT OR DELETE ON persons
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION party_assert_exactly_one_subtype();

CREATE CONSTRAINT TRIGGER organizations_exactly_one_subtype
AFTER INSERT OR DELETE ON organizations
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION party_assert_exactly_one_subtype();

COMMIT;
