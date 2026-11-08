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
  IF number_of_tables <> 27
     OR to_regclass('public.oauth_grants') IS NULL
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
     OR NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 30 AND NOT dirty)
     OR NOT EXISTS (SELECT 1 FROM information_schema.columns
           WHERE table_schema='public' AND table_name='accounts'
             AND column_name='password_reset_version') THEN
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

# A database with application data but no migration version must not be adopted.
# The migrate driver may create schema_migrations; this must not apply version 1.
psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_unversioned_legacy_test'
psql -h 127.0.0.1 -U root -d bdspro_unversioned_legacy_test -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE accounts (id bigint PRIMARY KEY);
INSERT INTO accounts(id) VALUES (42);
SQL
if output=$(BDS__DATABASE__DB_SOURCE='postgres://root:secret@127.0.0.1:5432/bdspro_unversioned_legacy_test?sslmode=disable' make migrateup 2>&1); then
  echo 'Unversioned database with application data was incorrectly accepted' >&2
  exit 1
fi
if [[ "$output" != *'unversioned database contains'* ]]; then
  echo "Migration was rejected for the wrong reason: $output" >&2
  exit 1
fi
psql -h 127.0.0.1 -U root -d bdspro_unversioned_legacy_test -v ON_ERROR_STOP=1 <<'SQL'
DO $check$
DECLARE marked boolean := false;
BEGIN
  IF (SELECT count(*) FROM accounts WHERE id = 42) <> 1
     OR to_regclass('public.parties') IS NOT NULL THEN
    RAISE EXCEPTION 'migration modified an unversioned database with existing data';
  END IF;
  IF to_regclass('public.schema_migrations') IS NOT NULL THEN
    EXECUTE 'SELECT EXISTS (SELECT 1 FROM public.schema_migrations)' INTO marked;
    IF marked THEN
      RAISE EXCEPTION 'migration assigned a version to the unversioned database';
    END IF;
  END IF;
END $check$;
SQL

# A view without any application table must also block adoption.
# This catches guards that look only at information_schema.tables.
psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_unversioned_view_test'
psql -h 127.0.0.1 -U root -d bdspro_unversioned_view_test -v ON_ERROR_STOP=1 -c 'CREATE VIEW legacy_view AS SELECT 42::bigint AS id'
if output=$(BDS__DATABASE__DB_SOURCE='postgres://root:secret@127.0.0.1:5432/bdspro_unversioned_view_test?sslmode=disable' make migrateup 2>&1); then
  echo 'Unversioned database with existing view was incorrectly accepted' >&2
  exit 1
fi
if [[ "$output" != *'unversioned database contains'* ]]; then
  echo "View-only database was rejected for the wrong reason: $output" >&2
  exit 1
fi
psql -h 127.0.0.1 -U root -d bdspro_unversioned_view_test -v ON_ERROR_STOP=1 <<'SQL'
DO $check$
BEGIN
  IF to_regclass('public.legacy_view') IS NULL
     OR to_regclass('public.parties') IS NOT NULL THEN
    RAISE EXCEPTION 'migration modified the pre-existing view-only database';
  END IF;
END $check$;
SQL
