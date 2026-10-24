#!/bin/sh
# Guard committed migration history against edits or deletion in a change.
# Only compares Git trees; no SQL, Docker, network or working-tree writes.
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: sh scripts/check-migration-history.sh BASE_SHA HEAD_SHA" >&2
  exit 2
fi

base=$1
head=$2

git cat-file -e "$base^{commit}" || { echo "unknown base commit: $base" >&2; exit 2; }
git cat-file -e "$head^{commit}" || { echo "unknown head commit: $head" >&2; exit 2; }

blocked=$(git diff --name-status --no-renames --diff-filter=CDMRTUXB "$base" "$head" -- db/migration/)
if [ -n "$blocked" ]; then
  printf 'FAIL: existing migration files must not be changed or removed:\n%s\n' "$blocked" >&2
  echo "Create new forward migration files instead." >&2
  exit 1
fi

echo "PASS: existing migration history unchanged (new files allowed)"
