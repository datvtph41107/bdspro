-- name: CreateTeamForOwner :one
INSERT INTO teams (organization_id, name, description)
SELECT sqlc.arg(organization_id)::bigint, btrim(sqlc.arg(name)::text), sqlc.arg(description)::text
WHERE EXISTS (
    SELECT 1 FROM organization_members m
    JOIN accounts a ON a.person_id = m.person_id
    WHERE m.organization_id = sqlc.arg(organization_id)
      AND a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.role = 'owner' AND m.ended_at IS NULL
  FOR SHARE OF m
)
RETURNING id, organization_id, name, description, created_at, updated_at;

-- name: GetTeamForAccount :one
SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at
FROM teams t
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  );

-- name: ListTeamsForAccount :many
SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at
FROM teams t
WHERE t.organization_id = sqlc.arg(organization_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
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
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
    FOR SHARE OF m
  )
RETURNING t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at;

-- name: AddTeamUserForOwner :one
WITH owner_scope AS MATERIALIZED (
  SELECT m.organization_id
  FROM teams t
  JOIN organization_members m ON m.organization_id = t.organization_id
  JOIN accounts a ON a.person_id = m.person_id
  WHERE t.id = sqlc.arg(team_id)
    AND a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
    AND m.role = 'owner' AND m.ended_at IS NULL
  FOR SHARE OF m
),
recipient_scope AS MATERIALIZED (
  SELECT m.organization_id, m.person_id
  FROM owner_scope owner
  JOIN organization_members m ON m.organization_id = owner.organization_id
  JOIN accounts recipient ON recipient.person_id=m.person_id AND recipient.status='active'
  WHERE m.person_id = sqlc.arg(person_id) AND m.ended_at IS NULL
  FOR SHARE OF m
)
INSERT INTO team_members (team_id, person_id)
SELECT t.id, recipient.person_id
FROM recipient_scope recipient
JOIN teams t ON t.organization_id = recipient.organization_id
WHERE t.id = sqlc.arg(team_id)
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
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
    FOR SHARE OF m
  );

-- name: ListTeamUsersForAccount :many
SELECT tm.team_id, tm.person_id, tm.joined_at
FROM team_members tm
JOIN teams t ON t.id = tm.team_id
WHERE t.id = sqlc.arg(team_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL AND m.role='owner'
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
SET revoked_at = clock_timestamp()
WHERE team_id = sqlc.arg(team_id)
  AND invited_email = sqlc.arg(invited_email)
  AND accepted_at IS NULL AND revoked_at IS NULL
  AND (
    expires_at <= clock_timestamp() OR team_invites.recipient_membership_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM organization_members original WHERE original.id=team_invites.recipient_membership_id AND original.ended_at IS NULL)
    OR EXISTS (
      SELECT 1 FROM teams t
      JOIN organization_members m ON m.organization_id = t.organization_id
      JOIN accounts recipient ON recipient.person_id = m.person_id
      WHERE t.id = team_invites.team_id
        AND m.ended_at IS NULL
        AND recipient.is_email_verified = TRUE
        AND lower(btrim(recipient.email)) = lower(btrim(team_invites.invited_email))
        AND team_invites.created_at < m.joined_at
    )
  );

-- name: CreateTeamInviteForOwner :one
WITH owner_scope AS MATERIALIZED (
  SELECT t.id AS team_id, t.organization_id
  FROM teams t
  JOIN organization_members m ON m.organization_id = t.organization_id
  JOIN accounts a ON a.person_id = m.person_id
  WHERE t.id = sqlc.arg(team_id)
    AND a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
    AND m.role = 'owner' AND m.ended_at IS NULL
  FOR SHARE OF m
),
recipient_scope AS MATERIALIZED (
  SELECT owner.team_id, recipient.person_id, recipient.id AS recipient_account_id, member.id AS recipient_membership_id
  FROM owner_scope owner
  JOIN organization_members member ON member.organization_id = owner.organization_id
  JOIN accounts recipient ON recipient.person_id = member.person_id
  WHERE recipient.is_email_verified = TRUE AND recipient.status='active'
    AND lower(btrim(recipient.email)) = lower(btrim(sqlc.arg(invited_email)::text))
    AND member.ended_at IS NULL
    AND NOT EXISTS (
      SELECT 1 FROM team_members tm
      WHERE tm.team_id = owner.team_id AND tm.person_id = recipient.person_id
    )
  FOR SHARE OF member
)
INSERT INTO team_invites (id, team_id, invited_email, invited_by_account_id, token_hash, expires_at, recipient_account_id, recipient_membership_id)
SELECT sqlc.arg(id)::uuid, recipient.team_id, lower(btrim(sqlc.arg(invited_email)::text)),
       sqlc.arg(actor_account_id)::bigint, sqlc.arg(token_hash)::bytea, sqlc.arg(expires_at)::timestamptz, recipient.recipient_account_id, recipient.recipient_membership_id
FROM recipient_scope recipient
RETURNING id, team_id, invited_email, invited_by_account_id, token_hash,
          expires_at, accepted_at, revoked_at, created_at;

-- name: ConsumeTeamInvite :one
WITH locked_invite AS MATERIALIZED (
    SELECT id, expires_at FROM team_invites
    WHERE token_hash=sqlc.arg(token_hash)::bytea
      AND recipient_account_id=sqlc.arg(account_id)::bigint
      AND accepted_at IS NULL AND revoked_at IS NULL
    FOR UPDATE
)
UPDATE team_invites AS ti
SET accepted_at = clock_timestamp()
FROM locked_invite AS locked, teams AS t, accounts AS a, organization_members AS m
WHERE ti.id = locked.id AND locked.expires_at > clock_timestamp()
  AND t.id = ti.team_id
  AND a.id = sqlc.arg(account_id)::bigint AND a.is_email_verified = TRUE AND a.status='active'
  AND ti.recipient_account_id=a.id AND ti.recipient_membership_id=m.id
  AND lower(btrim(a.email)) = lower(btrim(ti.invited_email))
  AND m.organization_id = t.organization_id
  AND m.person_id = a.person_id AND m.ended_at IS NULL
  AND ti.created_at >= m.joined_at
RETURNING ti.team_id;

-- name: InsertInvitedTeamUser :one
INSERT INTO team_members (team_id, person_id)
SELECT t.id, a.person_id FROM teams AS t
JOIN accounts AS a ON a.id = sqlc.arg(account_id)
JOIN organization_members AS m
  ON m.organization_id = t.organization_id
 AND m.person_id = a.person_id AND m.ended_at IS NULL
WHERE t.id = sqlc.arg(team_id)
FOR SHARE OF m
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
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id
      AND m.role = 'owner' AND m.ended_at IS NULL
    FOR SHARE OF m
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
    WHERE recipient.is_email_verified = TRUE AND recipient.status='active'
      AND ti.recipient_account_id=recipient.id AND ti.recipient_membership_id=m.id
      AND lower(btrim(recipient.email)) = lower(btrim(ti.invited_email))
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
      AND ti.created_at >= m.joined_at
  );

-- name: CountTeamsForAccount :one
SELECT count(*)::bigint
FROM teams t
WHERE t.organization_id = sqlc.arg(organization_id)
  AND EXISTS (
    SELECT 1 FROM accounts a
    JOIN organization_members m ON m.person_id = a.person_id
    WHERE a.id = sqlc.arg(actor_account_id) AND a.status = 'active'
      AND m.organization_id = t.organization_id AND m.ended_at IS NULL
  );

-- name: FindTeamInviteOrganization :one
SELECT t.organization_id,ti.id FROM team_invites ti JOIN teams t ON t.id=ti.team_id
WHERE ti.token_hash=sqlc.arg(token_hash)::bytea AND ti.recipient_account_id=sqlc.arg(account_id)::bigint
AND ti.accepted_at IS NULL AND ti.revoked_at IS NULL AND ti.expires_at>clock_timestamp();

-- name: CountTeamUsersForOwner :one
SELECT count(*)::bigint FROM team_members tm JOIN teams t ON t.id=tm.team_id
WHERE tm.team_id=sqlc.arg(team_id)::bigint AND EXISTS (
    SELECT 1 FROM organization_members m JOIN accounts a ON a.person_id=m.person_id
    WHERE a.id=sqlc.arg(actor_account_id)::bigint AND a.status='active'
    AND m.organization_id=t.organization_id AND m.role='owner' AND m.ended_at IS NULL
) AND EXISTS (
    SELECT 1 FROM organization_members m WHERE m.organization_id=t.organization_id
    AND m.person_id=tm.person_id AND m.ended_at IS NULL
);

-- name: GetOwnTeamMembership :one
SELECT tm.team_id,tm.person_id,tm.joined_at FROM team_members tm
JOIN teams t ON t.id=tm.team_id JOIN accounts a ON a.person_id=tm.person_id
JOIN organization_members m ON m.organization_id=t.organization_id AND m.person_id=a.person_id
WHERE tm.team_id=sqlc.arg(team_id)::bigint AND a.id=sqlc.arg(account_id)::bigint
AND a.status='active' AND m.ended_at IS NULL;

-- name: CancelUndeliveredTeamInvite :execrows
UPDATE team_invites SET revoked_at=clock_timestamp()
WHERE id=$1 AND accepted_at IS NULL AND revoked_at IS NULL;
