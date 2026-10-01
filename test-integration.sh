#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  pwd
)"

cd "$ROOT_DIR"

ENV_FILE="${BDS_ENV_FILE:-$ROOT_DIR/.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "environment file not found: $ENV_FILE" >&2
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"

readonly TEST_DB_NAME="bdspro_test"

if [[ "$POSTGRES_DB" == "$TEST_DB_NAME" ]]; then
  echo "refusing to continue: development database and test database are both '$TEST_DB_NAME'" >&2
  exit 1
fi

unset TEST_DATABASE_URL || true

echo "safety gate passed"
echo "development database: $POSTGRES_DB"
echo "integration database: $TEST_DB_NAME"
