# Engineering workflow — human and agent

This directory explains how to work, verify and learn in BDSPro. Its audience is a developer or agent joining without access to earlier conversations. Source at a pinned commit and observed runtime behavior outrank documentation when they disagree.

- [Task and acceptance contract](task-contract.md): the G0–G4 sequence, evidence tiers, review boundaries and a reusable PR/task record.
- [Verify BDSPro skill](../../.cursor/skills/verify-bdspro/SKILL.md): launch disposable dependencies, run verified HTTP acceptance and keep local evidence.
- [Capability map](capability-map.md): how to discover code paths and authoritative tests for the first protected capabilities.
- [Current source architecture](../architecture/README.md), [development and testing](../testing.md) and [source ownership](../architecture/source-ownership.md): current and historical engineering guidance.
- [Live session authority](../architecture/g1-live-session-authority.md), [organization transaction ordering](../architecture/g1-organization-transaction-order.md) and [organization naming](../architecture/orgs-gitea-decision.md): capability-specific constraints.

A successful instruction file only proves that agents can *read* a policy. A working check proves a narrow property. A reviewed end-to-end behavior proves more. Never promote one type of evidence into another without a test.

## How a new engineer should start

1. Check the repository HEAD, branch, uncommitted edits and open PRs. Read the smallest relevant set of sources, then trace one real request from route through persistence and response.
2. Identify what the current tests demonstrate and what they do not (especially races, outages, workload and security boundaries).
3. Write down one falsifiable hypothesis and its minimal test. Implement only after determining which boundary owns the behavior.
4. Review findings with a second person or independent verification lane when risk warrants it; capture the actual artifacts.
5. When a lesson repeats, prefer a type restriction, DB constraint, checker or CI test over another paragraph of prompt.

This is intentionally a small starting point, not a framework or a promise of production readiness. Verification skills, controlled agent evals and autonomy promotion must earn their place through measured tasks and independently verified evidence.
