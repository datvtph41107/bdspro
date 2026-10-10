-- ONLY for a disposable/dev rebaseline, not a populated production DB.
-- This file reverses 000001. All data in these three tables will be removed.
BEGIN;

DROP TRIGGER parties_exactly_one_subtype ON parties;
DROP FUNCTION party_require_exactly_one_subtype();

DROP TRIGGER persons_immutable_identity ON persons;
DROP TRIGGER organizations_immutable_identity ON organizations;
DROP FUNCTION party_prevent_subtype_change();

DROP TRIGGER persons_lock_on_insert ON persons;
DROP TRIGGER organizations_lock_on_insert ON organizations;
DROP FUNCTION party_lock_for_subtype_insert();

DROP TABLE persons;
DROP TABLE organizations;
DROP TABLE parties;

COMMIT;
