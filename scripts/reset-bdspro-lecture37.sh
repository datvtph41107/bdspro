#!/usr/bin/env bash
set -euo pipefail

# This script intentionally replaces ONLY the local Docker PostgreSQL
# database named "bdspro", after verifying a complete backup.
# Execute from repository root after fetching the Lecture 37 dev source.

if [[ "${CONFIRM_RESET_BDSPRO:-}" != "RESET_BDSPRO_TO_LECTURE37" ]]; then
  echo "REFUSED: set CONFIRM_RESET_BDSPRO=RESET_BDSPRO_TO_LECTURE37 to acknowledge replacement of all data in bdspro." >&2
  exit 2
fi

container="${POSTGRES_CONTAINER:-postgres}"
target="bdspro"
dsn="postgres://root:secret@127.0.0.1:5432/bdspro?sslmode=disable"

for tool in docker migrate; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing required CLI: $tool" >&2
    exit 3
  fi
done

if [[ ! -f "db/migration/000001_party_account_session.up.sql" ]]; then
  echo "Run this from the Lecture 37 BDSPro repository root." >&2
  exit 3
fi

if [[ "$(git branch --show-current)" != "dev" ]]; then
  echo "REFUSED: switch to dev before resetting the database." >&2
  exit 3
fi

if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
  echo "REFUSED: commit/stash local tracked changes before resetting database." >&2
  exit 3
fi

docker exec "$container" pg_isready -U root -d postgres

exists="$(docker exec "$container" psql -U root -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname='bdspro'")"
backup=""
if [[ "$exists" == "1" ]]; then
  umask 077
  backup_dir="${HOME}/bdspro-db-backups"
  mkdir -p "$backup_dir"
  timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
  backup="$backup_dir/bdspro-before-lecture37-${timestamp}-$$.dump"

  echo "Backing up existing database bdspro to: $backup"
  if ! docker exec "$container" pg_dump -U root -d "$target" --format=custom > "$backup"; then
    rm -f "$backup"
    echo "Backup failed. Database has NOT been modified." >&2
    exit 4
  fi
  if [[ ! -s "$backup" ]] || ! docker exec -i "$container" pg_restore --list < "$backup" >/dev/null; then
    echo "Backup verification failed. Database has NOT been modified." >&2
    exit 4
  fi
  echo "Backup checked. Existing application connections to bdspro will be terminated."
  docker exec "$container" psql -U root -d postgres -v ON_ERROR_STOP=1 \
    -c 'DROP DATABASE bdspro WITH (FORCE);'
fi

docker exec "$container" createdb -U root -O root "$target"
if ! migrate -path db/migration -database "$dsn" -verbose up; then
  echo "Migration failed. Restore your verified backup from: $backup" >&2
  exit 5
fi

status="$(docker exec "$container" psql -U root -d "$target" -Atqc "
SELECT CASE WHEN
 (SELECT COUNT(*) FROM information_schema.tables
  WHERE table_schema='public' AND table_name IN ('parties','persons','accounts','sessions'))=4
 AND EXISTS (SELECT 1 FROM information_schema.columns
  WHERE table_schema='public' AND table_name='sessions'
        AND column_name='id' AND data_type='uuid')
 AND EXISTS (SELECT 1 FROM information_schema.columns
  WHERE table_schema='public' AND table_name='accounts'
        AND column_name='password_hash')
 AND EXISTS (SELECT 1 FROM schema_migrations WHERE version=1 AND dirty=false)
 AND NOT EXISTS (SELECT 1 FROM information_schema.tables
  WHERE table_schema='public' AND table_name='organizations')
 THEN 'OK' ELSE 'SCHEMA_MISMATCH' END;")"

if [[ "$status" != "OK" ]]; then
  echo "Schema check failed: $status. Refer to backup: $backup" >&2
  exit 6
fi

echo "SUCCESS: bdspro is now the Lecture 37 schema."
if [[ -n "$backup" ]]; then echo "Previous database backup: $backup"; fi
echo "Run: go test ./api ./token ./util && make server"
