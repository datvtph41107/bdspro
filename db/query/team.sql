-- name: CreateTeamForOwner :one
INSERT INTO teams (organization_id, name, description)
SELECT sqlc.arg(organization_id)::bigint, btrim(sqlc.arg(name)::text), sqlc.arg(description)::text
WHERE EXISTS (
    SELECT 1 FROM organization_members m
    JOIN accounts a ON a.person_id = m.person_id
    WHERE m.organization_id = sqlc.arg(organization_id)
      AND a.id = sqlc.arg(actor_account_id)
      AND m.role = 'owner' AND m.ended_at IS NULL
)
RETURNING id, organization_id, name, description, created_at, updated_at;

-- name: GetTeamForAccount :one
SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at
FROM teams t
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  );

-- name: ListTeamsForAccount :many
SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at
FROM teams t
WHERE t.organization_id = sqlc.arg(organization_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  )
ORDER BY t.id
LIMIT sqlc.arg(take)::integer OFFSET sqlc.arg(skip)::integer;

-- name: RenameTeamForOwner :one
UPDATE teams AS t
SET name = btrim(sqlc.arg(name)::text), updated_at = now()
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
  )
RETURNING t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at;

-- name: AddTeamUserForOwner :one
INSERT INTO team_members (team_id, person_id)
SELECT t.id, sqlc.arg(person_id)::bigint FROM teams t
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
  )
  AND EXISTS (
    SELECT 1 FROM organization_members m
    WHERE m.person_id = sqlc.arg(person_id)
      AND m.organization_id = t.organization_id
      AND m.ended_at IS NULL
  )
RETURNING team_id, person_id, joined_at;

-- name: RemoveTeamUserForOwner :execrows
DELETE FROM team_members AS tm
USING teams AS t
WHERE tm.team_id = t.id
  AND t.id = sqlc.arg(team_id)
  AND tm.person_id = sqlc.arg(person_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
  );

-- name: ListTeamUsersForAccount :many
SELECT tm.team_id, tm.person_id, tm.joined_at
FROM team_members tm
JOIN teams t ON t.id = tm.team_id
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  )
  AND EXISTS (
    SELECT 1 FROM organization_members m
    WHERE m.organization_id = t.organization_id
      AND m.person_id = tm.person_id AND m.ended_at IS NULL
  )
ORDER BY tm.joined_at, tm.person_id
LIMIT sqlc.arg(take)::integer OFFSET sqlc.arg(skip)::integer;

-- name: RevokeExpiredTeamInvites :exec
UPDATE team_invites
SET revoked_at = now()
WHERE team_id = sqlc.arg(team_id)
  AND invited_email = sqlc.arg(invited_email)
  AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at <= now();

-- name: CreateTeamInviteForOwner :one
INSERT INTO team_invites (id, team_id, invited_email, invited_by_account_id, token_hash, expires_at)
SELECT sqlc.arg(id)::uuid, t.id, lower(btrim(sqlc.arg(invited_email)::text)), sqlc.arg(actor_account_id)::bigint,
       sqlc.arg(token_hash)::bytea, sqlc.arg(expires_at)::timestamptz
FROM teams t
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
  )
  -- A Team invitation is only valid for a verified current Organization member.
  AND EXISTS (
    SELECT 1 FROM accounts recipient
    JOIN organization_members member ON member.person_id = recipient.person_id
    WHERE recipient.is_email_verified = TRUE
      AND lower(btrim(recipient.email)) = lower(btrim(sqlc.arg(invited_email)::text))
      AND member.organization_id = t.organization_id AND member.ended_at IS NULL
      AND NOT EXISTS (
        SELECT 1 FROM team_members tm
        WHERE tm.team_id = t.id AND tm.person_id = recipient.person_id
      )
  )
RETURNING id, team_id, invited_email, invited_by_account_id, token_hash,
          expires_at, accepted_at, revoked_at, created_at;

-- name: ConsumeTeamInvite :one
UPDATE team_invites AS ti
SET accepted_at = now()
FROM teams AS t, accounts AS a, organization_members AS m
WHERE ti.token_hash = sqlc.arg(token_hash)
  AND ti.accepted_at IS NULL AND ti.revoked_at IS NULL AND ti.expires_at > now()
  AND t.id = ti.team_id
  AND a.id = sqlc.arg(account_id) AND a.is_email_verified = TRUE
  AND lower(btrim(a.email)) = lower(btrim(ti.invited_email))
  AND m.organization_id = t.organization_id
  AND m.person_id = a.person_id AND m.ended_at IS NULL
RETURNING ti.team_id;

-- name: InsertInvitedTeamUser :one
INSERT INTO team_members (team_id, person_id)
SELECT t.id, a.person_id FROM teams AS t
JOIN accounts AS a ON a.id = sqlc.arg(account_id)
JOIN organization_members AS m
  ON m.organization_id = t.organization_id
 AND m.person_id = a.person_id AND m.ended_at IS NULL
WHERE t.id = sqlc.arg(team_id)
RETURNING team_id, person_id, joined_at;

-- name: RevokeTeamInviteForOwner :execrows
UPDATE team_invites AS ti
SET revoked_at = now()
FROM teams AS t
WHERE ti.id = sqlc.arg(invite_id)
  AND ti.team_id = t.id
  AND t.id = sqlc.arg(team_id)
  AND ti.accepted_at IS NULL AND ti.revoked_at IS NULL
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id)
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
  );

-- name: GetTeamInviteForDelivery :one
SELECT ti.invited_email, ti.token_hash, ti.expires_at,
       t.name AS team_name, o.name AS organization_name
FROM team_invites ti
JOIN teams t ON t.id = ti.team_id
JOIN organizations o ON o.party_id = t.organization_id
WHERE ti.id = sqlc.arg(invite_id)::uuid
  AND ti.accepted_at IS NULL AND ti.revoked_at IS NULL AND ti.expires_at > now()
  AND EXISTS (
    SELECT 1 FROM accounts recipient
    JOIN organization_members m ON m.person_id = recipient.person_id
    WHERE recipient.is_email_verified = TRUE
      AND lower(btrim(recipient.email)) = lower(btrim(ti.invited_email))
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  );
