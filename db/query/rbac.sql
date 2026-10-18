-- name: GrantInitialMemberRemovalPermission :exec
INSERT INTO organization_role_permissions (role_id, permission_code)
VALUES ($1, 'members.remove')
ON CONFLICT DO NOTHING;

-- name: LockOrganizationForSourceWrite :one
SELECT party_id FROM organizations
WHERE party_id = $1 AND archived_at IS NULL
FOR SHARE;

-- name: EndAuthorizedOrganizationMembership :execrows
UPDATE organization_memberships AS target SET ended_at = now()
WHERE target.organization_id = sqlc.arg(organization_id)::bigint
  AND target.id = sqlc.arg(membership_id)::bigint
  AND target.ended_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM organization_primary_administrators pa
    WHERE pa.organization_id = target.organization_id AND pa.membership_id = target.id
  )
  AND EXISTS (
    SELECT 1 FROM accounts actor
    JOIN organization_memberships m
      ON m.organization_id = target.organization_id AND m.person_id = actor.person_id
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE actor.id = sqlc.arg(actor_account_id)::bigint
      AND actor.closed_at IS NULL
      AND m.ended_at IS NULL AND m.joined_at <= now()
      AND ra.ended_at IS NULL AND ra.assigned_at <= now()
      AND rp.permission_code = 'members.remove'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.ended_at IS NULL AND b.blocked_at <= now()
      )
  );

-- name: EndOrganizationMembershipRoles :execrows
UPDATE organization_role_assignments SET ended_at = now()
WHERE organization_id = sqlc.arg(organization_id)::bigint
  AND membership_id = sqlc.arg(membership_id)::bigint AND ended_at IS NULL;
