-- Revert only in disposable test databases. This removes grants made after UP.
BEGIN;
DELETE FROM organization_role_permissions WHERE permission_code = 'members.remove';
DELETE FROM permissions WHERE code = 'members.remove';
COMMIT;
