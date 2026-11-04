#!/usr/bin/env bash
set -euo pipefail

if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Refusing OAuth2 migration tests outside disposable mode" >&2
  exit 1
fi

psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1   -c 'CREATE DATABASE bdspro_oauth2_upgrade_test'
URL='postgres://root:secret@127.0.0.1:5432/bdspro_oauth2_upgrade_test?sslmode=disable'

# Start at the last schema version before OAuth2 credential storage.
migrate -path db/migration -database "$URL" up 12
psql -h 127.0.0.1 -U root -d bdspro_oauth2_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties (id) OVERRIDING SYSTEM VALUE VALUES (991201), (991202);
INSERT INTO persons (party_id) VALUES (991201), (991202);
INSERT INTO accounts (id, person_id, email) OVERRIDING SYSTEM VALUE
VALUES (991201, 991201, 'legacy-oidc-r26@example.test'),
       (991202, 991202, 'new-oidc-r26@example.test');
INSERT INTO google_identities (google_sub, account_id)
VALUES ('legacy-sub-r26', 991201);
SQL

# Backfill, temporary bridge, rename to credentials and archive legacy links.
migrate -path db/migration -database "$URL" up 4
psql -h 127.0.0.1 -U root -d bdspro_oauth2_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF to_regclass('public.oauth2_credentials') IS NULL
     OR to_regclass('public.oauth2_identities') IS NOT NULL
     OR to_regclass('public.google_identities') IS NOT NULL
     OR to_regclass('public.oauth2_credentials_archive') IS NULL
     OR NOT EXISTS (
         SELECT 1 FROM oauth2_credentials c
         JOIN oauth2_credentials_archive a
           ON a.subject = c.subject AND a.account_id = c.account_id
         WHERE c.issuer = 'https://accounts.google.com'
           AND c.subject = 'legacy-sub-r26' AND c.account_id = 991201
     ) THEN
    RAISE EXCEPTION 'OAuth2 credential migration did not retain legacy account linkage';
  END IF;
END $audit$;
INSERT INTO oauth2_credentials (issuer, subject, account_id)
VALUES ('https://accounts.google.com', 'new-sub-r26', 991202);
SQL

# Roll back archive step and ensure identities created by the new binary
# are available to the old reader without changing account IDs.
migrate -path db/migration -database "$URL" down 1
psql -h 127.0.0.1 -U root -d bdspro_oauth2_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF to_regclass('public.google_identities') IS NULL
     OR (SELECT count(*) FROM google_identities
         WHERE (google_sub = 'legacy-sub-r26' AND account_id = 991201)
            OR (google_sub = 'new-sub-r26' AND account_id = 991202)) <> 2
  THEN
    RAISE EXCEPTION 'OAuth2 archive rollback did not reconcile credentials';
  END IF;
END $audit$;
SQL

migrate -path db/migration -database "$URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_oauth2_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF (SELECT count(*) FROM oauth2_credentials
      WHERE issuer = 'https://accounts.google.com'
        AND subject IN ('legacy-sub-r26', 'new-sub-r26')) <> 2
     OR (SELECT count(*) FROM oauth2_credentials_archive
         WHERE subject IN ('legacy-sub-r26', 'new-sub-r26')) <> 2
  THEN
    RAISE EXCEPTION 'OAuth2 credential re-upgrade lost account links';
  END IF;
END $audit$;
SQL
