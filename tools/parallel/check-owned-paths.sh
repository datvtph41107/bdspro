#!/usr/bin/env bash
# Advisory path-ownership check for BDSPro parallel coding lanes.
# Usage: bash tools/parallel/check-owned-paths.sh A origin/dev
set -euo pipefail

if (( $# < 1 || $# > 2 )); then
  echo 'usage: bash tools/parallel/check-owned-paths.sh {A|B|C|D|0} [base-ref]' >&2
  exit 2
fi
lane="$1"
base="${2:-origin/dev}"
case "$lane" in A|B|C|D|0) ;; *) echo "unknown lane: $lane" >&2; exit 2 ;; esac
if ! git rev-parse --verify "${base}^{commit}" >/dev/null 2>&1; then
  echo "base ref not found: $base; fetch the expected base before checking" >&2
  exit 2
fi
invalid=0
seen=0
while IFS= read -r -d '' path; do
  seen=$((seen + 1))
  allowed=0
  case "$lane" in
    A) case "$path" in modules/web/*|services/context/*) allowed=1 ;; esac ;;
    B) case "$path" in models/*|services/user/*) allowed=1 ;; esac ;;
    C) case "$path" in cmd/*|modules/setting/*|modules/log/*|modules/cache/*|modules/queue/*|modules/graceful/*) allowed=1 ;; esac ;;
    D) case "$path" in routers/api/v1/*|routers/web/oauth/*|services/auth/*|modules/auth/*) allowed=1 ;; esac ;;
    0) case "$path" in docs/architecture/parallel-work/*|tools/parallel/*|.github/workflows/*|routers/init.go) allowed=1 ;; esac ;;
  esac
  if (( ! allowed )); then
    printf 'OUT-OF-SCOPE lane=%s path=%s\n' "$lane" "$path" >&2
    invalid=$((invalid + 1))
  fi
done < <(git diff --name-only --diff-filter=ACMR -z "${base}...HEAD")
if (( invalid > 0 )); then
  printf 'FAIL: %d out-of-scope path(s) among %d changed files. Request a coordinator-owned file lease, not an unreviewed exception.\n' "$invalid" "$seen" >&2
  exit 1
fi
printf 'PASS: %d changed path(s) within lane %s allowed write set (base %s).\n' "$seen" "$lane" "$base"
