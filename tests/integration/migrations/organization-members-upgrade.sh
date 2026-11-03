#!/usr/bin/env bash
set -euo pipefail
if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Refusing to create/write databases without BDS_TEST_MODE=disposable" >&2
  exit 1
fi
psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_org_upgrade_test'
ORG_URL='postgres://root:secret@127.0.0.1:5432/bdspro_org_upgrade_test?sslmode=disable'
migrate -path db/migration -database "$ORG_URL" up 10
psql -h 127.0.0.1 -U root -d bdspro_org_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties (id) OVERRIDING SYSTEM VALUE VALUES (901), (902);
INSERT INTO persons (party_id) VALUES (901), (902);
INSERT INTO organizations (party_id, name) VALUES (901, 'Upgrade fixture');
INSERT INTO organization_memberships (organization_id, person_id, role, ended_at)
VALUES (901, 901, 'owner', NULL), (901, 902, 'member', now());
SQL
migrate -path db/migration -database "$ORG_URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_org_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF to_regclass('public.organization_members') IS NULL
    OR to_regclass('public.organization_memberships') IS NOT NULL
    OR to_regclass('public.organization_members_one_open_uq') IS NULL
    OR (SELECT count(*) FROM organization_members WHERE organization_id = 901) <> 2
    OR (SELECT count(*) FROM organization_members WHERE role = 'owner' AND ended_at IS NULL) <> 1
    OR (SELECT count(*) FROM organization_members WHERE role = 'member' AND ended_at IS NOT NULL) <> 1
  THEN
    RAISE EXCEPTION 'organization member rename did not preserve data or constraints';
  END IF;
END $audit$;
SQL
migrate -path db/migration -database "$ORG_URL" down 1
psql -h 127.0.0.1 -U root -d bdspro_org_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF to_regclass('public.organization_memberships') IS NULL
    OR to_regclass('public.organization_members') IS NOT NULL
    OR (SELECT count(*) FROM organization_memberships WHERE organization_id = 901) <> 2
  THEN
    RAISE EXCEPTION 'organization member rename is not reversible';
  END IF;
END $audit$;
SQL
