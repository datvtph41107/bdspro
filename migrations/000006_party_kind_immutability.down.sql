-- Reverts only the new kind-immutability guard; Party data is preserved.
-- LAB / controlled rollback: removing this guard reopens a prohibited identity transition.
BEGIN;
DROP TRIGGER party_kind_immutable ON parties;
DROP FUNCTION reject_party_kind_change();
COMMIT;
