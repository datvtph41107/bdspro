-- Reverts only CORE-01 additions. Run after all later dependent migrations
-- have been reverted. Never run down on a production dataset casually.
BEGIN;

DROP TRIGGER organizations_exactly_one_subtype ON organizations;
DROP TRIGGER persons_exactly_one_subtype ON persons;
DROP TRIGGER parties_exactly_one_subtype ON parties;

DROP TRIGGER organizations_lock_party ON organizations;
DROP TRIGGER persons_lock_party ON persons;

DROP FUNCTION party_assert_exactly_one_subtype();
DROP FUNCTION party_lock_for_subtype_write();

DROP TABLE organizations;
DROP TABLE persons;
DROP TABLE parties;

COMMIT;
