#!/bin/sh
set -eu

# Verify a complete, immutable Git-tree snapshot. This proves file identities
# at the pinned baseline; it does NOT claim every behavior is production-ready.
cd "$(dirname "$0")/.."

baseline=6f47f4971fba49ba8ce9479adbfb9739c244f620
snapshot=docs/engineering/source-baseline-6f47f49.tree
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

git cat-file -e "$baseline^{commit}"
git ls-tree -r --full-tree "$baseline" | LC_ALL=C sort > "$tmp/actual"
LC_ALL=C sort "$snapshot" > "$tmp/expected"

if ! diff -u "$tmp/expected" "$tmp/actual"; then
    echo "Core baseline does not match its pinned Git objects" >&2
    exit 1
fi

entries="$(wc -l < "$tmp/actual" | tr -d ' ')"
go_files="$(git ls-tree -r --name-only "$baseline" | grep -c '\.go$' || true)"
test_files="$(git ls-tree -r --name-only "$baseline" | grep -c '_test\.go$' || true)"
sql_files="$(git ls-tree -r --name-only "$baseline" | grep -c '\.sql$' || true)"
printf 'Verified immutable source baseline: %s\n' "$baseline"
printf 'Tracked files: %s, Go: %s, Go tests: %s, SQL: %s\n' \
    "$entries" "$go_files" "$test_files" "$sql_files"
printf 'Current HEAD: %s\n' "$(git rev-parse HEAD)"

# CI tests, race, smoke and runtime checks are separate evidence, not proved
# by Git object existence. See docs/engineering/core-source-proof.md.
