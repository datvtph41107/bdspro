#!/usr/bin/env bash
set -euo pipefail
if [[ "${BDS_TEST_MODE:-}" != "disposable" ]]; then
  echo "Refusing to create/write databases without BDS_TEST_MODE=disposable" >&2
  exit 1
fi
psql -h 127.0.0.1 -U root -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE bdspro_upgrade_test'
UPGRADE_URL='postgres://root:secret@127.0.0.1:5432/bdspro_upgrade_test?sslmode=disable'
migrate -path db/migration -database "$UPGRADE_URL" up 6
psql -h 127.0.0.1 -U root -d bdspro_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO parties(id) OVERRIDING SYSTEM VALUE VALUES (1), (2);
INSERT INTO persons(party_id) VALUES (1), (2);
INSERT INTO accounts(id, person_id, email, password_hash) OVERRIDING SYSTEM VALUE
VALUES (1, 1, 'old1@example.test', 'legacy-hash-1'), (2, 2, 'old2@example.test', 'legacy-hash-2');
INSERT INTO verification_outbox(id, account_id, queued_at) OVERRIDING SYSTEM VALUE
VALUES (19, 1, NULL), (21, 2, now());
SQL
migrate -path db/migration -database "$UPGRADE_URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF (SELECT count(*) FROM job_outbox) <> 2
      OR (SELECT count(*) FROM job_outbox WHERE queued_at IS NULL) <> 1
      OR (SELECT count(*) FROM job_outbox WHERE queued_at IS NOT NULL) <> 1
      OR (SELECT task_type FROM job_outbox WHERE id = 19) <> 'task:send_verify_email'
      OR (SELECT payload->>'account_id' FROM job_outbox WHERE id = 19) <> '1'
      OR (SELECT task_id FROM job_outbox WHERE id = 19) <> 'verify-email-outbox-19'
      OR to_regclass('public.verification_outbox') IS NOT NULL
      OR nextval(pg_get_serial_sequence('job_outbox','id')) <= 21 THEN
    RAISE EXCEPTION 'outbox migration did not preserve pending task state';
  END IF;
END $audit$;
SQL
migrate -path db/migration -database "$UPGRADE_URL" up 1
psql -h 127.0.0.1 -U root -d bdspro_upgrade_test -v ON_ERROR_STOP=1 <<'SQL'
DO $audit$
BEGIN
  IF (SELECT count(*) FROM password_credentials) <> 2
      OR (SELECT password_hash FROM password_credentials WHERE account_id = 1) <> 'legacy-hash-1'
      OR EXISTS (
          SELECT 1 FROM information_schema.columns
          WHERE table_name = 'accounts' AND column_name = 'password_hash'
      ) THEN
    RAISE EXCEPTION 'password credential migration lost an existing hash';
  END IF;
END $audit$;
SQL
