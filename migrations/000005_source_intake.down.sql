-- LAB ONLY: drops source content and its price history.
-- Do not use as a data-preserving production rollback.
BEGIN;
DROP TABLE source_price_reports;
DROP TABLE source_intakes;
-- Keep permission codes while any role references them. Permission
-- catalogue cleanup is a separately reviewed policy/migration.
DELETE FROM permissions AS p
WHERE p.code IN ('sources.create', 'sources.read', 'sources.price_report')
  AND NOT EXISTS (
    SELECT 1 FROM organization_role_permissions AS rp
    WHERE rp.permission_code = p.code
  );
COMMIT;
