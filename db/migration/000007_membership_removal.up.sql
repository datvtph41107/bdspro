-- Introduce a distinct revocation authority; do not reuse invitation permission.
BEGIN;
INSERT INTO permissions (code) VALUES ('members.remove') ON CONFLICT DO NOTHING;
-- Existing primary administrators retain administrative authority after upgrade.
INSERT INTO organization_role_permissions (role_id, permission_code)
SELECT ra.role_id, 'members.remove'
FROM organization_primary_administrators pa
JOIN organization_role_assignments ra
  ON ra.organization_id = pa.organization_id AND ra.membership_id = pa.membership_id
JOIN organization_roles r
  ON r.id = ra.role_id AND r.organization_id = pa.organization_id
WHERE ra.ended_at IS NULL AND r.name = 'Workspace Administrator'
ON CONFLICT DO NOTHING;
COMMIT;
