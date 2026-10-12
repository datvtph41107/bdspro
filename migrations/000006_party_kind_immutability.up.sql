-- CORE-01 follow-up: Party kind is a durable identity property, not a role.
-- A transaction must not replace a PERSON with an ORGANIZATION at the same Party ID.
-- Do not rewrite 000001: existing migrated databases need a forward-only migration.
BEGIN;

CREATE FUNCTION reject_party_kind_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  IF OLD.kind IS DISTINCT FROM NEW.kind THEN
    RAISE EXCEPTION 'Party kind is immutable (party_id=%): % -> %',
      OLD.id, OLD.kind, NEW.kind
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER party_kind_immutable
BEFORE UPDATE ON parties
FOR EACH ROW
EXECUTE FUNCTION reject_party_kind_change();

COMMIT;
