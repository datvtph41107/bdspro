#!/usr/bin/env bash
set -euo pipefail

if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Refusing migration test outside disposable mode" >&2
  exit 1
fi

psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_profile_upgrade_test'
URL='postgres://root:secret@127.0.0.1:5432/bdspro_profile_upgrade_test?sslmode=disable'
migrate -path db/migration -database "$URL" up 16
psql -h 127.0.0.1 -U root -d bdspro_profile_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties(id) OVERRIDING SYSTEM VALUE VALUES (1001), (1002), (1003);
INSERT INTO persons(party_id) VALUES (1001), (1002), (1003);
INSERT INTO accounts(id, person_id, email) OVERRIDING SYSTEM VALUE
VALUES (1001,1001,'before-profile-a@example.test'), (1002,1002,'before-profile-b@example.test');
SQL
migrate -path db/migration -database "$URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_profile_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF (SELECT count(*) FROM person_profiles) <> 3
      OR (SELECT count(*) FROM account_settings) <> 2
      OR (SELECT count(*) FROM person_profiles WHERE display_name='' AND first_name='' AND last_name='') <> 3
      OR (SELECT count(*) FROM account_settings WHERE locale='vi' AND timezone='Asia/Ho_Chi_Minh' AND currency='VND') <> 2
      OR (SELECT count(*) FROM accounts) <> 2 THEN
    RAISE EXCEPTION 'person profile/account preferences backfill lost legacy data';
  END IF;
END $audit$;
SQL
migrate -path db/migration -database "$URL" down 1
migrate -path db/migration -database "$URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_profile_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF (SELECT count(*) FROM person_profiles) <> 3
      OR (SELECT count(*) FROM account_settings) <> 2 THEN
    RAISE EXCEPTION 'profile rollback/upgrade did not restore existing accounts';
  END IF;
END $audit$;
SQL
