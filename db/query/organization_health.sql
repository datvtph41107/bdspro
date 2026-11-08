-- name: ListOrganizationsWithoutActiveOwner :many
-- Operator diagnostic, not a hot-path authorization check.
-- An existing organization can lose its last owner through privileged raw SQL.
-- Returning at most 21 IDs supports capped output without scanning all rows
-- into application memory. Do not change this to COUNT(*) + unbounded listing.
SELECT o.party_id
FROM organizations AS o
WHERE NOT EXISTS (
  SELECT 1
  FROM organization_members AS m
  WHERE m.organization_id = o.party_id
    AND m.role = 'owner'
    AND m.ended_at IS NULL
)
ORDER BY o.party_id
LIMIT 21;
