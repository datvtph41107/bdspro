---
name: bdspro-pr-verification
description: Use for independent BDSPro pull-request review, pre-merge verification, agent handoff, exact-HEAD evidence checks or investigating a claimed completed change.
---

# Evidence-led PR and release verification

1. Fetch **live** `dev`, PR base/head SHA, changed files, merged/draft status and current CI runs. Distinguish GitHub metadata from stale PR title/body or historical docs.
2. Read root `AGENTS.md`, the owner/lease rules and appropriate source-investigation or change skill; check each changed file's ownership. A permission to push is not permission to merge.
3. For every central claim: source permalink, affected caller, invariant, code path and test that can refute it. If comparing with Gitea/Discourse, pin commits and state *which mechanism* transfers, not package similarity.
4. Review adversarial cases: concurrency, authorization bypass, nil/invalid inputs, nested transaction, cancellation, partial external failure, crash between DB commit and queue enqueue, migration compatibility and startup/shutdown as applicable.
5. Verify actual CI status **on the final PR HEAD**; explicitly label missing/skipped tests, infra unavailable, insufficient coverage and pending status. A green earlier commit is not accepted. If CI is green but changes are semantically unverified, mark them unverified.
6. Report a small table for each concern: **claim | evidence | PASS / FAIL / UNKNOWN / N/A | risk / remedy**. Give precise root cause, not a superficial lint checklist.
7. Handoff as a reviewable PR comment or document: base/HEAD, changed paths, sources, commands/logs, observed outcomes, unresolved risk, proposed action and rollback. No direct `dev` merge unless existing governance and user permissions explicitly authorize it.

A review process should continuously convert repeated human comments into deterministic tests or scripts when those are actually needed; don't add unconsumed gates.
