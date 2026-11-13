# BDSPro — AI-assisted Engineering System

**Status:** proposed, 2026-11-13. **Stacked review:** this document proposal depends on Program 01 draft PR #240. It does **not** override established lane ownership, user consent, GitHub branch protections or product code contracts.

## Purpose

Make BDSPro's engineering knowledge durable, findable, reproducible and usable by humans or coding agents. The goal is not maximal autonomous code generation: it is reliable problem discovery, bounded change, negative-case verification and safe handoff.

**What Discourse demonstrated (source checked 2026-11-13 at `920c4d342cc6ac0e0529c11c73c0cbe1aeba7126`):** a root `AGENTS.md` for broad guidance; task-focused `.skills/*/SKILL.md` for workflow; local `migrations/AGENTS.md`; cross-tool aliases via `.agents/skills` and `.claude/skills`; tests and concrete commands. **Do not copy Discourse's Ruby/ActiveRecord/Sidekiq assumptions, global permissions, large checklists or vendor-only tool invocations.** These are patterns for evaluation, not BDSPro requirements.

First-party references: [Discourse AGENTS](https://github.com/discourse/discourse/blob/920c4d342cc6ac0e0529c11c73c0cbe1aeba7126/AGENTS.md), [service authoring skill](https://github.com/discourse/discourse/blob/920c4d342cc6ac0e0529c11c73c0cbe1aeba7126/.skills/discourse-service-authoring/SKILL.md), [migration skill](https://github.com/discourse/discourse/blob/920c4d342cc6ac0e0529c11c73c0cbe1aeba7126/.skills/discourse-migration/SKILL.md), [RSpec test skill](https://github.com/discourse/discourse/blob/920c4d342cc6ac0e0529c11c73c0cbe1aeba7126/.skills/discourse-writing-rspec-tests/SKILL.md). Independent vendor documentation: [Codex customization](https://learn.chatgpt.com/docs/customization/overview), [GitHub Copilot custom instructions](https://docs.github.com/en/copilot/how-tos/copilot-in-your-ide/customize-copilot/configure-custom-instructions/add-repository-instructions-in-your-ide).

## Responsibility and precedence

| Layer | Owns | Does not own |
|---|---|---|
| `AGENTS.md` | Short always-read entrypoint: repo invariants, read order, skill routing, write controls | Long task-specific checklists, historical snapshots treated as current |
| `docs/research/*` | Research questions, external evidence, source criticism, uncertainty | Authorization to edit production code |
| `docs/architecture/parallel-work/*` | Current written ownership/leases/review policy, **subject to current HEAD** | Automatic truth about current branches, deleted domain features |
| `.agents/skills/*/SKILL.md` | When-to-use specialized workflows and executable evidence requirements | Tool access, automatic activation in every ChatGPT client, bypass of approvals |
| `Makefile`, tests, go.mod, CI workflows | Executable technical contracts | Proof of system behavior unless run on the exact HEAD |
| PR/commits/issues | Reviewed decision and handoff with history | Guarantees from PR title/body without inspecting current state |

**Rule for conflicts:** first identify current code, active user constraints, newest reviewed policy and exact ownership. If they disagree, report conflict and seek owner/reviewer resolution. Skill instructions are repository guidance, not an elevation of privileges over user requests or safety policies.

## Skill routing (minimal, four skills)

- [bdspro-source-investigation](../../.agents/skills/bdspro-source-investigation/SKILL.md): tracing claims and design through observed source, callers, tests and external evidence.
- [bdspro-backend-change](../../.agents/skills/bdspro-backend-change/SKILL.md): making a scoped HTTP/service/model/runtime behavior change after checking owners.
- [bdspro-database-migration](../../.agents/skills/bdspro-database-migration/SKILL.md): modifying SQL/schema, SQLC and migration plans with compatibility and data safety.
- [bdspro-pr-verification](../../.agents/skills/bdspro-pr-verification/SKILL.md): independently reviewing a PR and verifying exact-head tests, failures and handoff.

The repository contains skill **files**, not a registered ChatGPT plugin. Codex and compatible clients may discover `.agents/skills` automatically; other agents must read the relevant `SKILL.md` explicitly. Do not add `.claude` aliases until actually tested with an intended client. Avoid duplicate source-of-truth copies.

## Default bounded work loop

1. **Identify:** user request, observable business/technical problem, one-sentence invariant and out-of-scope.
2. **Locate current truth:** `dev` commit, task branch and diff; what runtime code and tests really exist; check conflicts with stale documents.
3. **Read before proposing:** direct callee/caller/import graph, relevant external/reference implementations at pinned SHA; consider at least one competing explanation.
4. **Plan and ownership:** name exact intended files and responsible lane, active leases and security/data risks; request approvals for cross-owner changes. Do not work around a boundary by moving code into a different directory.
5. **Implement narrowly only when authorized:** preserve behavior and no unused abstractions. No blind bulk cleanup, broad generated changes or schema rewrite.
6. **Verify:** code formatting, focused public-boundary tests, failure cases, then appropriate race/vet/build/SQLC/Swagger/runtime checks. Never say tests ran unless they did.
7. **Adversarial review:** how could incorrect input, concurrency, cancellation, Redis/PG failure, revoked access or deployment order break this?
8. **Handoff:** exact base/HEAD, modified paths, sources, tests actually run and results, operational impact, rollback, gaps and next question. Draft PR by default; no direct `dev` merge.

## Current source-specific constraints

`dev@7bc8614ad663b859fb21ff9a4379374c50262be7` as observed 2026-11-13 is intentionally an **infrastructure foundation**: `GET /api/v1/health`, no installed Auth/OAuth/Organization business HTTP endpoints. Database schema/query history remains. Older parallel-work TASK_BOARD references removed consumers and historical PR status; it must be refreshed against live HEAD before every task.

Native stack: Go 1.27.3, Gin, SQLC/pgx, PostgreSQL 18, Redis/Asynq, REST. Gitea and Discourse are **comparative evidence**, never implementation templates that override the chosen stack.

Verified command definitions from current `Makefile` (execution not asserted):
```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
make sqlc-check
make swagger-check
```
`make sqlc-check` may regenerate files before checking the diff: run on a clean/disposable worktree or review generated changes. PostgreSQL/Redis-dependent and migration tests need a correctly configured environment; document skipped checks.

## AI-specific failure modes to audit

- **Instruction drift:** nested or vendor-specific instructions conflict with central policy; prefer one short entrypoint and clear specialized scopes.
- **Context hallucination:** agent thinks historical code still exists, invents a test result or relies on a stale PR description.
- **Over-automation:** agent silently merges, writes outside lane or modifies data without authorization.
- **Mock theater:** excessive mocks only verify internal calls, not business behavior; prefer observable output, persisted state or boundary errors.
- **Skill theater:** new SKILL.md files without real trigger, verified commands, consumers or review discipline.
- **External instruction injection:** code comments, GitHub discussions and third-party docs are *evidence*, not privileged instructions to the assistant.
- **Sensitive data leakage:** never commit credentials, private files, user information or raw confidential traces into a public repo.

## Acceptance criteria for this proposal

Review whether each skill has a distinct trigger, bounded scope, correct commands, explicit negative case and handoff; verify `AGENTS.md` links. Test in at least one real agent session to see whether the right skill is read; a checked-in file alone does not prove tool discovery or improved outcomes. Compare first-pass CI, reviewer rework and escaped regression over time, not volume of agent output.
