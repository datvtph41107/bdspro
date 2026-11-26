-- name: ListOrganizationsWithoutActiveOwner :many
SELECT o.party_id
FROM organizations AS o
WHERE o.party_id > sqlc.arg(after_id)::bigint
  AND NOT EXISTS (
  SELECT 1
  FROM organization_members AS m
  WHERE m.organization_id = o.party_id
    AND m.role = 'owner'
    AND m.ended_at IS NULL
)
ORDER BY o.party_id
LIMIT 21;
