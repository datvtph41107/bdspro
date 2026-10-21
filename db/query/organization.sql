-- name: CreateOrganization :one
INSERT INTO organizations (party_id, name)
VALUES ($1, $2)
RETURNING party_id, name, created_at;

-- name: CreateOrganizationMembership :one
INSERT INTO organization_memberships (organization_id, person_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, person_id, role, joined_at, ended_at;

-- name: ListOrganizationsForPerson :many
SELECT o.party_id, o.name, o.created_at
FROM organizations o
JOIN organization_memberships m ON m.organization_id = o.party_id
WHERE m.person_id = $1 AND m.ended_at IS NULL
ORDER BY o.party_id;

-- name: AddOrganizationMember :one
INSERT INTO organization_memberships (organization_id, person_id, role)
SELECT sqlc.arg(organization_id), sqlc.arg(person_id), 'member'
WHERE EXISTS (
  SELECT 1
  FROM organization_memberships owner
  JOIN accounts a ON a.person_id = owner.person_id
  WHERE owner.organization_id = sqlc.arg(organization_id)
    AND a.id = sqlc.arg(actor_account_id)
    AND owner.role = 'owner'
    AND owner.ended_at IS NULL
)
RETURNING id, organization_id, person_id, role, joined_at, ended_at;
