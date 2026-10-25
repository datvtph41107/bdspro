#!/bin/sh
set -eu

# Inspect the exact committed Git tree at HEAD, never an inferred directory
# listing or ignored local files. --list prints one record per tracked blob.
# The registry maps source paths to an owner; neither mode proves behavior.
cd "$(dirname "$0")/.."

case "${1:---summary}" in
    --summary|--list) mode="${1:---summary}" ;;
    *) echo "usage: sh scripts/check-core-current-source.sh [--summary|--list]" >&2; exit 2 ;;
esac

git cat-file -e 'HEAD^{commit}'
head_commit="$(git rev-parse HEAD)"
head_tree="$(git rev-parse 'HEAD^{tree}')"
registry=docs/engineering/source-ownership.tsv
test -f "$registry"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT HUP INT TERM
git ls-tree -r --full-tree HEAD > "$tmp"

awk -F '\t' -v registry="$registry" -v mode="$mode" \
    -v commit="$head_commit" -v tree="$head_tree" '
BEGIN {
    while ((getline line < registry) > 0) {
        split(line, row, "\t")
        if (row[1] == "prefix") continue
        if (row[1] == "" || row[2] == "" || row[3] == "") {
            print "Invalid source owner registry entry: " line > "/dev/stderr"
            bad = 1
            continue
        }
        if (seen[row[1]]++) {
            print "Duplicate ownership prefix: " row[1] > "/dev/stderr"
            bad = 1
        }
        prefix[++registry_count] = row[1]
        owner[registry_count] = row[2]
        purpose[registry_count] = row[3]
    }
    close(registry)
    if (registry_count == 0) bad = 1
    if (mode == "--summary") {
        print "HEAD commit: " commit
        print "HEAD tree:   " tree
    } else {
        print "owner\tmode\tblob_sha\tpath\tresponsibility"
    }
}
{
    if (NF != 2) {
        print "Unrecognized Git tree entry: " $0 > "/dev/stderr"
        bad = 1
        next
    }
    n = split($1, meta, " ")
    if (n != 3 || meta[2] != "blob") {
        print "Unexpected non-blob entry: " $0 > "/dev/stderr"
        bad = 1
        next
    }

    path = $2
    best = 0
    longest = -1
    for (i = 1; i <= registry_count; ++i) {
        p = prefix[i]
        matches = (p == "(root)" && index(path, "/") == 0) ||
            (p != "(root)" && (path == p || index(path, p "/") == 1))
        if (matches && length(p) > longest) {
            best = i
            longest = length(p)
        }
    }
    if (best == 0) {
        print "UNOWNED tracked path: " path > "/dev/stderr"
        bad = 1
        next
    }

    files++
    per_owner[best]++
    if (path ~ /\.go$/) go_files++
    if (path ~ /_test\.go$/) test_files++
    if (path ~ /\.sql$/) sql_files++
    if (path ~ /^pb\/.*\.go$/ || path == "doc/statik/statik.go" ||
        path ~ /^db\/sqlc\/.*\.sql\.go$/ ||
        path == "db/sqlc/models.go" || path == "db/sqlc/querier.go" ||
        path ~ /^db\/mock\// || path ~ /^worker\/mock\//) generated_files++

    if (mode == "--list")
        printf "%s\t%s\t%s\t%s\t%s\n", owner[best], meta[1], meta[3], path, purpose[best]
}
END {
    if (mode == "--summary") {
        printf "Owned tracked files: %d (Go: %d, tests: %d, SQL: %d, known generated: %d)\n", \
            files, go_files, test_files, sql_files, generated_files
        for (i = 1; i <= registry_count; ++i) {
            if (per_owner[i] > 0)
                printf "  %-27s %3d  [%s]\n", owner[i], per_owner[i], prefix[i]
        }
    }
    if (bad || files == 0) exit 1
}
' "$tmp"
