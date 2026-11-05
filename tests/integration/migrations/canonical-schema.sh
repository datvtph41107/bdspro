#!/usr/bin/env bash
set -euo pipefail
if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Migration acceptance requires BDS_TEST_MODE=disposable" >&2
  exit 1
fi
psql -h 127.0.0.1 -U root -d bdspro_test -v ON_ERROR_STOP=1 <<'SQL'
DO $check$
DECLARE number_of_tables integer;
BEGIN
  SELECT count(*) INTO number_of_tables
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
    AND table_name <> 'schema_migrations';
  IF number_of_tables <> 26
     OR to_regclass('public.oauth_clients') IS NULL
     OR to_regclass('public.oauth_auth_codes') IS NULL
     OR to_regclass('public.oauth_access_tokens') IS NULL
     OR to_regclass('public.oauth_refresh_tokens') IS NULL
     OR to_regclass('public.oauth2_credentials') IS NULL
     OR to_regclass('public.google_identities') IS NOT NULL
     OR to_regclass('public.oauth2_identities') IS NOT NULL
     OR to_regclass('public.oauth2_credentials_archive') IS NOT NULL
     OR to_regclass('public.organization_memberships') IS NOT NULL
     OR to_regclass('public.verification_outbox') IS NOT NULL
     OR EXISTS (SELECT 1 FROM pg_trigger WHERE NOT tgisinternal)
     OR NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 1 AND NOT dirty) THEN
    RAISE EXCEPTION 'canonical fresh schema differs from contract (tables=%)', number_of_tables;
  END IF;
END $check$;
SQL

psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_prebaseline_test'
psql -h 127.0.0.1 -U root -d bdspro_prebaseline_test -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE schema_migrations(version bigint NOT NULL, dirty boolean NOT NULL);
INSERT INTO schema_migrations(version, dirty) VALUES (21, false);
SQL
if BDS__DATABASE__DB_SOURCE='postgres://root:secret@127.0.0.1:5432/bdspro_prebaseline_test?sslmode=disable' make migrateup; then
  echo 'Pre-baseline migration v21 was incorrectly accepted' >&2
  exit 1
fi
