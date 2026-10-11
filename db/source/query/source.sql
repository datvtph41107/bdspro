-- CORE-05 sqlc queries. Never trust org/account context from client payloads:
-- actor_account_id must come from a verified session in the HTTP adapter.

-- name: InsertSourceIntake :one
INSERT INTO source_intakes (
  organization_id, recorded_by_account_id, source_channel_code,
  source_label, reported_area_m2, area_basis_code, received_at, summary,
  capture_request_key, capture_request_hash
)
SELECT
  o.party_id, a.id, sqlc.arg(channel)::text,
  NULLIF(btrim(sqlc.arg(label)::text), ''),
  NULLIF(sqlc.arg(area_m2)::text, '')::numeric,
  NULLIF(btrim(sqlc.arg(area_basis)::text), ''),
  NULLIF(sqlc.arg(received_at)::text, '')::timestamptz,
  NULLIF(btrim(sqlc.arg(summary)::text), ''),
  sqlc.arg(request_key)::text, sqlc.arg(request_hash)::text
FROM organizations o
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
WHERE o.party_id = sqlc.arg(organization_id)::bigint
  AND o.archived_at IS NULL
  AND a.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = o.party_id AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.create'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  )
  AND (
    NOT sqlc.arg(has_initial_price)::boolean
    OR EXISTS (
      SELECT 1 FROM organization_memberships m
      JOIN organization_role_assignments ra
        ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
      JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
      WHERE m.organization_id = o.party_id AND m.person_id = a.person_id
        AND m.joined_at <= NOW() AND m.ended_at IS NULL
        AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
        AND rp.permission_code = 'sources.price_report'
        AND NOT EXISTS (
          SELECT 1 FROM organization_membership_blocks b
          WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
            AND b.ended_at IS NULL
        )
    )
  )
ON CONFLICT ON CONSTRAINT source_intakes_request_uq DO NOTHING
RETURNING id;

-- name: FindAuthorizedCaptureReplay :one
SELECT si.id, si.capture_request_hash AS request_hash
FROM source_intakes si
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
JOIN organizations o ON o.party_id = si.organization_id
WHERE si.organization_id = sqlc.arg(organization_id)::bigint
  AND si.recorded_by_account_id = a.id
  AND si.capture_request_key = sqlc.arg(request_key)::text
  AND a.closed_at IS NULL AND o.archived_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = si.organization_id
      AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.create'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  );

-- name: InsertPriceReport :one
INSERT INTO source_price_reports (
  organization_id, intake_id, recorded_by_account_id,
  amount, currency_code, price_basis_code, source_basis_code, statement_at, note,
  request_key, request_hash
)
SELECT
  si.organization_id, si.id, a.id,
  (sqlc.arg(amount)::text)::numeric,
  sqlc.arg(currency)::text,
  NULLIF(btrim(sqlc.arg(price_basis)::text), ''),
  NULLIF(btrim(sqlc.arg(source_basis)::text), ''),
  NULLIF(sqlc.arg(statement_at)::text, '')::timestamptz,
  NULLIF(btrim(sqlc.arg(note)::text), ''),
  sqlc.arg(price_request_key)::text,
  sqlc.arg(price_request_hash)::text
FROM source_intakes si
JOIN organizations o ON o.party_id = si.organization_id
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
WHERE si.id = sqlc.arg(intake_id)::bigint
  AND si.organization_id = sqlc.arg(organization_id)::bigint
  AND si.archived_at IS NULL AND o.archived_at IS NULL AND a.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = si.organization_id AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.price_report'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  )
ON CONFLICT ON CONSTRAINT source_price_reports_request_uq DO NOTHING
RETURNING id;

-- name: GetAuthorizedSource :one
SELECT
  si.id, si.organization_id, si.recorded_by_account_id,
  si.source_channel_code,
  COALESCE(si.source_label, '')::text AS source_label,
  COALESCE(si.reported_area_m2::text, '')::text AS reported_area_m2,
  COALESCE(si.area_basis_code, '')::text AS area_basis_code,
  COALESCE(si.summary, '')::text AS summary,
  si.version, si.recorded_at, si.received_at
FROM source_intakes si
JOIN organizations o ON o.party_id = si.organization_id
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
WHERE si.id = sqlc.arg(intake_id)::bigint
  AND si.organization_id = sqlc.arg(organization_id)::bigint
  AND si.archived_at IS NULL AND o.archived_at IS NULL AND a.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = si.organization_id AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.read'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  );

-- name: ListAuthorizedPriceReports :many
SELECT
  spr.id, spr.intake_id, spr.recorded_by_account_id,
  spr.amount::text AS amount, spr.currency_code,
  COALESCE(spr.price_basis_code, '')::text AS price_basis_code,
  COALESCE(spr.source_basis_code, '')::text AS source_basis_code,
  COALESCE(spr.note, '')::text AS note,
  spr.recorded_at, spr.statement_at
FROM source_price_reports spr
JOIN source_intakes si
  ON si.organization_id = spr.organization_id AND si.id = spr.intake_id
JOIN organizations o ON o.party_id = si.organization_id
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
WHERE spr.organization_id = sqlc.arg(organization_id)::bigint
  AND spr.intake_id = sqlc.arg(intake_id)::bigint
  AND si.archived_at IS NULL AND o.archived_at IS NULL AND a.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = si.organization_id AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.read'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  )
ORDER BY spr.recorded_at, spr.id;


-- name: FindAuthorizedPriceReplay :one
SELECT spr.id, COALESCE(spr.request_hash, '')::text AS request_hash
FROM source_price_reports spr
JOIN source_intakes si
  ON si.id = spr.intake_id AND si.organization_id = spr.organization_id
JOIN organizations o ON o.party_id = si.organization_id
JOIN accounts a ON a.id = sqlc.arg(actor_account_id)::bigint
WHERE spr.organization_id = sqlc.arg(organization_id)::bigint
  AND spr.intake_id = sqlc.arg(intake_id)::bigint
  AND spr.recorded_by_account_id = a.id
  AND spr.request_key = sqlc.arg(price_request_key)::text
  AND si.archived_at IS NULL AND o.archived_at IS NULL AND a.closed_at IS NULL
  AND EXISTS (
    SELECT 1 FROM organization_memberships m
    JOIN organization_role_assignments ra
      ON ra.organization_id = m.organization_id AND ra.membership_id = m.id
    JOIN organization_roles role ON role.id = ra.role_id AND role.organization_id = m.organization_id
    JOIN organization_role_permissions rp ON rp.role_id = ra.role_id
    WHERE m.organization_id = si.organization_id AND m.person_id = a.person_id
      AND m.joined_at <= NOW() AND m.ended_at IS NULL
      AND ra.assigned_at <= NOW() AND ra.ended_at IS NULL
      AND rp.permission_code = 'sources.price_report'
      AND NOT EXISTS (
        SELECT 1 FROM organization_membership_blocks b
        WHERE b.membership_id = m.id AND b.blocked_at <= NOW()
          AND b.ended_at IS NULL
      )
  );
