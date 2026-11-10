---
name: bdspro-gitea-parallel-engineering
description: Separate BDSPro × Gitea implementation across chat contexts with strict file ownership, verifiable handoffs and integration gates.
---

# Parallel Engineering Skill — Workstream Playbook

**Repository-local reference, not an automatically installed ChatGPT skill.** Explicitly read it with the specialist task in every new chat.

## On entry
1. Confirm assigned role **A/B/C/D/0** and actual branch.
2. Read [README](./README.md) → [OWNERSHIP_AND_CONTRACTS](./OWNERSHIP_AND_CONTRACTS.md) → [TRUST_AND_VERIFICATION](./TRUST_AND_VERIFICATION.md) → [INTEGRATION_PROTOCOL](./INTEGRATION_PROTOCOL.md) → **only your specialist task**.
3. Fetch current `dev`, assigned branch HEAD/PR, CI and worktree diff; do not infer completed work from an old transcript.
4. Fetch Gitea source **and call sites**, compare source location, names, imports, signatures, runtime state, behavior, error semantics, cleanup. Pin Gitea blob SHA in report.
5. Write a one-paragraph task scope: intended files, exact invariant, what is explicitly **not** in scope, and acceptance test.

## Before committing
1. Keep changes inside lane's write paths. Run `bash tools/parallel/check-owned-paths.sh A origin/dev` (substitute lane) once the script exists on the worker branch; until then read the file in docs branch and run the equivalent check. **A's historical PR #218 already includes cross-owner paths, so this advisory check will flag historical changes; obtain a documented migration exception instead of pretending the whole PR is clean.**
2. No new unused modules, copied folder scaffolds, additional handler/DB engine, random renames, or tests deleted to make build green.
3. Run gofmt, focused tests and source-specific checks; keep final full CI green.
4. Perform an adversarial case: what edge input/concurrency/failure would violate your claim?
5. Compare changed files against Gitea actual patterns and BDSPro neighboring package usage.
6. If cross-owner path is needed, stop and submit RFC in PR. Only coordinator can approve a lease or integration patch.

## Handoff
- PR URL, exact base/head commit, actual changed paths, pinned Gitea refs.
- Preserved and changed public interfaces.
- Focused test log and final head CI URL/step conclusions.
- One adversarial regression case, known risks, rollback and blocking dependencies.
- Use [HANDOFF_TEMPLATE.md](./HANDOFF_TEMPLATE.md). Source is authoritative if handoff text gets stale.

## Do not
- Write to `dev` directly or merge autonomously.
- Push to another lane's branch.
- Assert background work or cross-chat communication.
- Fix another team's CI by modifying its files without a lease.
- Claim "identical to Gitea" merely from names: Gin/SQLC are intentionally different.
- Advance to next phase because a previous workflow was green. Test the final HEAD.

## Advisory path script
`bash tools/parallel/check-owned-paths.sh B origin/dev`

- Exit 0: file paths allowed.
- Exit 1: unapproved paths found (requires coordinator handling).
- Exit 2: invalid lane/base ref or usage.
- **This script checks only path prefixes.** It is not wired to a branch-protection rule by this documentation PR; it does not check source semantics, test execution or CI. Add enforced workflows after existing PR #218 is reconciled with the new write-ownership policy.
