-- name: CreateOrganization :one
INSERT INTO organizations (party_id, name)
VALUES ($1, $2)
RETURNING party_id, name, created_at;

-- name: CreateOrganizationMember :one
INSERT INTO organization_members (organization_id, person_id, role, owner_person_id)
VALUES ($1, $2, $3, CASE WHEN $3 = 'owner' THEN $2 ELSE NULL END)
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
  FOR SHARE OF owner
)
RETURNING id, organization_id, person_id, role, joined_at, ended_at;

-- name: LockOrganizationCreator :one
SELECT person_id
FROM accounts
WHERE id = sqlc.arg(actor_account_id)
FOR KEY SHARE;

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
    FOR SHARE OF m
  )
RETURNING o.party_id, o.name, o.created_at;

-- name: EndOrdinaryOrganizationMemberForOwner :one
UPDATE organization_members AS target
SET ended_at = clock_timestamp()
WHERE target.organization_id = sqlc.arg(organization_id)
  AND target.person_id = sqlc.arg(person_id)
  AND target.role = 'member'
  AND target.ended_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_members AS owner
    JOIN accounts AS a ON a.person_id = owner.person_id
    WHERE owner.organization_id = target.organization_id
      AND a.id = sqlc.arg(actor_account_id)
      AND owner.role = 'owner' AND owner.ended_at IS NULL
    FOR SHARE OF owner
  )
RETURNING target.id, target.organization_id, target.person_id,
          target.role, target.joined_at, target.ended_at;

-- name: DeleteOrgMemberTeamLinks :execrows
DELETE FROM team_members AS tm
USING teams AS t
WHERE t.organization_id = sqlc.arg(organization_id)
  AND tm.person_id = sqlc.arg(person_id)
  AND tm.team_id = t.id;

-- name: LockOrganizationRoleChanges :one
-- The non-key row lock serializes all owner changes by organization, while
-- remaining compatible with FK KEY SHARE acquired by team/member inserts.
SELECT party_id
FROM organizations
WHERE party_id = sqlc.arg(organization_id)
FOR NO KEY UPDATE;

-- name: LockActingOrganizationOwner :one
-- Obtain the acting owner's person ID under a role-stable FOR SHARE lock.
-- The service holds the organization lock first; do not reverse this order.
SELECT m.person_id
FROM organization_members m
JOIN accounts a ON a.person_id = m.person_id
WHERE m.organization_id = sqlc.arg(organization_id)
  AND a.id = sqlc.arg(actor_account_id)
  AND m.role = 'owner' AND m.ended_at IS NULL
FOR SHARE OF m;

-- name: CountActiveOrganizationOwners :one
SELECT count(*)::bigint
FROM organization_members
WHERE organization_id = sqlc.arg(organization_id)
  AND role = 'owner' AND ended_at IS NULL;

-- name: PromoteOrganizationMember :one
-- This query is called only inside the role-service transaction after
-- LockOrganizationRoleChanges and LockActingOrganizationOwner.
-- A new owner must be an actual account with a verified contact, not an
-- orphan Person or a plain unverified account.
UPDATE organization_members AS m
SET role = 'owner', owner_person_id = m.person_id
WHERE m.organization_id = sqlc.arg(organization_id)
  AND m.person_id = sqlc.arg(person_id)
  AND m.role = 'member' AND m.ended_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM accounts recipient
    WHERE recipient.person_id = m.person_id
      AND (
        recipient.is_email_verified = TRUE OR EXISTS (
          SELECT 1 FROM phone_credentials pc
          WHERE pc.account_id = recipient.id AND pc.verified_at IS NOT NULL
        )
      )
    FOR KEY SHARE OF recipient
  )
RETURNING m.id, m.organization_id, m.person_id, m.role, m.joined_at, m.ended_at;

-- name: DemoteOrganizationOwner :one
-- The transaction owns both authorization and the last-owner policy.
-- Never use this query alone outside that transaction.
UPDATE organization_members AS m
SET role = 'member', owner_person_id = NULL
WHERE m.organization_id = sqlc.arg(organization_id)
  AND m.person_id = sqlc.arg(person_id)
  AND m.role = 'owner' AND m.ended_at IS NULL
RETURNING m.id, m.organization_id, m.person_id, m.role, m.joined_at, m.ended_at;
