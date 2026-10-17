-- Organization onboarding, reviewer locks, tenant membership and invitation authority.
-- Name queries by behavior; preserve authorization predicates and lock order.

-- name: SubmitOrganizationRegistration :one
INSERT INTO organization_registration_requests
  (requested_by_account_id, declared_name, declared_jurisdiction_code, declared_registration_code)
SELECT a.id, sqlc.arg(name)::text, sqlc.arg(jurisdiction)::text,
  NULLIF(sqlc.arg(registration)::text, '')
FROM accounts a
WHERE a.id = sqlc.arg(account_id)::bigint AND a.closed_at IS NULL
  AND EXISTS (SELECT 1 FROM account_emails ae WHERE ae.account_id = a.id AND ae.verified_at IS NOT NULL)
RETURNING id;

-- name: SubmitOrganizationClaim :one
INSERT INTO organization_claim_requests
  (organization_id, requested_by_account_id, evidence_reference)
SELECT o.party_id, a.id, sqlc.arg(evidence)::text
FROM organizations o JOIN accounts a ON a.id = sqlc.arg(account_id)::bigint
WHERE o.party_id = sqlc.arg(organization_id)::bigint AND o.archived_at IS NULL AND a.closed_at IS NULL
  AND EXISTS (SELECT 1 FROM account_emails ae WHERE ae.account_id = a.id AND ae.verified_at IS NOT NULL)
RETURNING id;

-- name: LockActivePlatformReviewer :one
SELECT r.account_id FROM platform_organization_reviewers r
JOIN accounts a ON a.id = r.account_id
WHERE r.account_id = $1 AND r.revoked_at IS NULL AND a.closed_at IS NULL
FOR SHARE OF r, a;

-- name: UpsertInitialOrganizationMembership :one
INSERT INTO organization_memberships (organization_id, person_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(person_id))
ON CONFLICT (organization_id, person_id) WHERE ended_at IS NULL
DO UPDATE SET person_id = EXCLUDED.person_id
RETURNING id;

-- name: CreateWorkspaceAdminRole :one
INSERT INTO organization_roles (organization_id, name)
VALUES ($1, 'Workspace Administrator') RETURNING id;

-- name: GrantInitialRolePermissions :exec
INSERT INTO organization_role_permissions (role_id, permission_code)
SELECT $1, code FROM permissions
WHERE code IN ('members.read','members.invite','sources.create','sources.read','sources.price_report');

-- name: CreateInitialRoleAssignment :exec
INSERT INTO organization_role_assignments (organization_id, membership_id, role_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(membership_id), sqlc.arg(role_id));

-- name: SetPrimaryAdministrator :exec
INSERT INTO organization_primary_administrators (organization_id, membership_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(membership_id));

-- name: GetPendingOrganizationRegistrationForUpdate :one
SELECT requested_by_account_id, declared_name, declared_jurisdiction_code,
       COALESCE(declared_registration_code, '')::text AS registration_code
FROM organization_registration_requests
WHERE id = $1 AND decision_code IS NULL FOR UPDATE;

-- name: CreateVerifiedRegistryIdentifier :exec
INSERT INTO organization_registry_identifiers
  (organization_id, jurisdiction_code, registration_code, verified_by_account_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(jurisdiction), sqlc.arg(registration), sqlc.arg(reviewer_id));

-- name: ResolveOrganizationRegistration :execrows
UPDATE organization_registration_requests
SET decision_code = 'APPROVED', resolved_at = now(),
    matched_organization_id = sqlc.arg(organization_id),
    reviewed_by_account_id = sqlc.arg(reviewer_id),
    review_reference = sqlc.arg(reference)
WHERE id = sqlc.arg(request_id) AND decision_code IS NULL;

-- name: GetPendingOrganizationClaimForUpdate :one
SELECT organization_id, requested_by_account_id
FROM organization_claim_requests
WHERE id = $1 AND decision_code IS NULL FOR UPDATE;

-- name: LockActiveOrganizationForUpdate :one
SELECT party_id FROM organizations
WHERE party_id = $1 AND archived_at IS NULL FOR UPDATE;

-- name: GetOrganizationPrimaryAdministrator :one
SELECT membership_id FROM organization_primary_administrators
WHERE organization_id = $1;

-- name: ResolveOrganizationClaim :execrows
UPDATE organization_claim_requests
SET decision_code = 'APPROVED', resolved_at = now(),
    reviewed_by_account_id = sqlc.arg(reviewer_id),
    review_reference = sqlc.arg(reference)
WHERE id = sqlc.arg(request_id) AND decision_code IS NULL;

-- name: CreateAuthorizedInvitation :one
INSERT INTO organization_invitations
  (organization_id, invitee_email, intended_role_id, invited_by_account_id, expires_at)
SELECT o.party_id, sqlc.arg(email)::text, role.id, a.id, now() + interval '7 days'
FROM organizations o
JOIN organization_roles role ON role.organization_id = o.party_id AND role.id = sqlc.arg(role_id)::bigint
JOIN accounts a ON a.id = sqlc.arg(account_id)::bigint
WHERE o.party_id = sqlc.arg(organization_id)::bigint
  AND o.archived_at IS NULL AND a.closed_at IS NULL AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = o.party_id AND m.person_id = a.person_id
      AND m.joined_at <= now() AND m.ended_at IS NULL
      AND ra.assigned_at <= now() AND ra.ended_at IS NULL
      AND rp.permission_code = 'members.invite'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.ended_at IS NULL AND b.blocked_at <= now()
      )
  )
RETURNING id;

-- name: AcceptPendingInvitation :one
UPDATE organization_invitations i
SET outcome = 'ACCEPTED', resolved_at = now()
WHERE i.id = sqlc.arg(invitation_id) AND i.outcome IS NULL AND i.expires_at > now()
  AND EXISTS (SELECT 1 FROM organizations o WHERE o.party_id = i.organization_id AND o.archived_at IS NULL)
  AND EXISTS (
    SELECT 1 FROM accounts a JOIN account_emails ae ON ae.account_id = a.id
    WHERE a.id = sqlc.arg(account_id) AND a.closed_at IS NULL
      AND ae.verified_at IS NOT NULL
      AND lower(btrim(ae.email)) = lower(btrim(i.invitee_email))
  )
RETURNING i.organization_id, i.intended_role_id;

-- name: CreateInvitationMembership :one
INSERT INTO organization_memberships (organization_id, person_id, originating_invitation_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(person_id), sqlc.arg(invitation_id))
RETURNING id;

-- name: AssignInvitationRole :exec
INSERT INTO organization_role_assignments (organization_id, membership_id, role_id)
VALUES (sqlc.arg(organization_id), sqlc.arg(membership_id), sqlc.arg(role_id));

-- name: ListAuthorizedMembers :many
SELECT target.id AS membership_id, target.person_id, target.joined_at
FROM organization_memberships target
JOIN organizations o ON o.party_id = target.organization_id
JOIN accounts actor ON actor.id = sqlc.arg(account_id)::bigint
WHERE target.organization_id = sqlc.arg(organization_id)::bigint
  AND target.ended_at IS NULL AND target.joined_at <= now()
  AND o.archived_at IS NULL AND actor.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = o.party_id AND m.person_id = actor.person_id
      AND m.ended_at IS NULL AND m.joined_at <= now()
      AND ra.ended_at IS NULL AND ra.assigned_at <= now()
      AND rp.permission_code = 'members.read'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.ended_at IS NULL AND b.blocked_at <= now()
      )
  )
ORDER BY target.id;
