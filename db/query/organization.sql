-- name: CreateOrganization :one
INSERT INTO organizations (party_id, name)
VALUES ($1, $2)
RETURNING party_id, name, created_at;

-- name: CreateOrganizationMember :one
INSERT INTO organization_members (organization_id, person_id, role)
VALUES ($1, $2, $3)
RETURNING id, organization_id, person_id, role, joined_at, ended_at;

-- name: ListOrganizationsForAccount :many
SELECT o.party_id, o.name, o.created_at,
       m.id AS membership_id, m.role, m.joined_at
FROM accounts a
JOIN organization_members m ON m.person_id = a.person_id
JOIN organizations o ON o.party_id = m.organization_id
WHERE a.id = sqlc.arg(account_id) AND m.ended_at IS NULL
ORDER BY o.party_id;

-- name: GetOrganizationForAccount :one
SELECT o.party_id, o.name, o.created_at,
       m.id AS membership_id, m.role, m.joined_at
FROM accounts a
JOIN organization_members m ON m.person_id = a.person_id
JOIN organizations o ON o.party_id = m.organization_id
WHERE a.id = sqlc.arg(account_id)
  AND o.party_id = sqlc.arg(organization_id)
  AND m.ended_at IS NULL;

-- name: ListOrganizationMembersForOwner :many
SELECT m.id, m.organization_id, m.person_id, m.role, m.joined_at
FROM organization_members m
WHERE m.organization_id = sqlc.arg(organization_id)
  AND m.ended_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM organization_members owner
    JOIN accounts a ON a.person_id = owner.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND owner.organization_id = m.organization_id
      AND owner.role = 'owner'
      AND owner.ended_at IS NULL
  )
ORDER BY m.joined_at, m.id;

-- name: AddOrganizationMember :one
INSERT INTO organization_members (organization_id, person_id, role)
SELECT sqlc.arg(organization_id), sqlc.arg(person_id), 'member'
WHERE EXISTS (
  SELECT 1
  FROM organization_members owner
  JOIN accounts a ON a.person_id = owner.person_id
  WHERE owner.organization_id = sqlc.arg(organization_id)
    AND a.id = sqlc.arg(actor_account_id)
    AND owner.role = 'owner'
    AND owner.ended_at IS NULL
)
RETURNING id, organization_id, person_id, role, joined_at, ended_at;

-- name: UpdateOrganization :one
UPDATE organizations AS o
SET name = COALESCE(sqlc.narg(name)::text, o.name)
WHERE o.party_id = sqlc.arg(organization_id)
  AND EXISTS (
    SELECT 1
    FROM organization_members AS m
    JOIN accounts AS a ON a.person_id = m.person_id
    WHERE m.organization_id = o.party_id
      AND a.id = sqlc.arg(actor_account_id)
      AND m.role = 'owner'
      AND m.ended_at IS NULL
  )
RETURNING o.party_id, o.name, o.created_at;
