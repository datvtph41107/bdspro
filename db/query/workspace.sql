-- List accessible workspaces using the currently authenticated Account's Person.
-- This is a navigation projection, not an action permission grant.
-- Every organization action MUST still check live membership and permission.
-- name: ListActiveAccountWorkspaces :many
SELECT o.party_id AS organization_id, o.name AS organization_name,
       m.id AS membership_id, m.joined_at
FROM accounts a
JOIN organization_memberships m ON m.person_id = a.person_id
JOIN organizations o ON o.party_id = m.organization_id
WHERE a.id = $1 AND a.closed_at IS NULL
  AND o.archived_at IS NULL
  AND m.ended_at IS NULL AND m.joined_at <= now()
  AND NOT EXISTS (
    SELECT 1 FROM organization_membership_blocks b
    WHERE b.membership_id = m.id
      AND b.ended_at IS NULL AND b.blocked_at <= now()
  )
  AND EXISTS (
    SELECT 1
    FROM organization_role_assignments ra
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE ra.organization_id = m.organization_id
      AND ra.membership_id = m.id
      AND ra.ended_at IS NULL AND ra.assigned_at <= now()
  )
ORDER BY lower(o.name), o.party_id;
