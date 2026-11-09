# BDSPro — engineering instructions for people and coding agents

This file is the shared entry point for Codex and Cursor; Claude Code imports it through [CLAUDE.md](CLAUDE.md).
It is guidance, not proof. Actual source, tests and CI results for the exact commit take precedence over stale documentation or chat memory.

## Start with the real repository

1. Inspect `git status`, current branch, `git rev-parse HEAD`, relevant PRs and the actual files before proposing a change. Distinguish local, remote and historical baselines.
2. Read the [developer index](docs/README.md), [engineering workflow](docs/engineering/README.md) and [capability map](docs/engineering/capability-map.md) only as relevant to the task. Verify every claimed path and call site.
3. Establish the problem and a falsifiable acceptance contract *before* implementation. Use the [task contract](docs/engineering/task-contract.md) for nontrivial work.
4. Treat [Gitea comparisons](docs/architecture/gitea-ownership-r01.md) as evidence about ownership and Go coding style, not a mandate to copy packages, Chi, XORM or names.

## Ownership and invariants

- Keep Go 1.27.3, Gin, PostgreSQL 18, pgx/SQLC, Redis and the existing CLI boundaries unless a separately approved change proves a need.
- `routers/` owns HTTP admission, binding and response; `services/` owns multi-step business operations; `models/` owns persistent domain behavior and transactions; `modules/` owns reusable technical mechanisms with real consumers. Inspect [source ownership](docs/architecture/source-ownership.md) and current callers; historical sections may be superseded.
- `db/query/` and `db/migration/` are SQL sources. `models/db/sqlc/` is generated-only; never hand-edit generated files. Do not introduce a second DB engine.
- Preserve the HTTP `/orgs` resource and full `organization` domain naming. Do not reintroduce `/organizations` silently. See the [accepted org naming decision](docs/architecture/orgs-gitea-decision.md).
- Never equate a signed JWT with an active session or request admission with transaction-serialized authorization. Consult the [live-session contract](docs/architecture/g1-live-session-authority.md) and test the affected authority path.
- Do not create packages, interfaces, helpers or compatibility layers solely to mimic Gitea. A new abstraction needs a caller, owned responsibility, failure semantics and a test.

## Evidence gates for every nontrivial change

- **G0 Ground:** Record exact baseline SHA, current behavior, source anchors, prior attempts and what is still unknown.
- **G1 Contract:** State actor/action/resource, happy and denied paths, invariants, transaction/outage behavior, compatibility and scope exclusions.
- **G2 Design:** Compare viable minimal options, including doing nothing. Challenge concurrency, rollback, retries, security, operations and complexity.
- **G3 Prove:** Reproduce a bug before fixing it where feasible. Run relevant unit/integration tests and exercise the real API/CLI when behavior changes; capture command, result and commit SHA. A self-report is not evidence.
- **G4 Accept:** Independent review and checks on the actual head, then update canonical docs and executable guardrails when lessons repeat. Report PASS, FAIL or INCONCLUSIVE; missing tests are not PASS.

For user-facing Organization and Session work, use the [verify-bdspro skill](.cursor/skills/verify-bdspro/SKILL.md) to run a disposable, real Gin/PostgreSQL/Redis acceptance lane; it supplements, not replaces, the existing tests and review. The first lane uses the in-process Gin HTTP adapter and is not a full production-network E2E proof.

Useful checks: `go test ./tests/architecture -count=1`; `make sqlc-check`; `make swagger-check`; `go test ./...`; `go test -race ./...`; `go vet ./...`. Run the applicable set and state what could not run. CI does not substitute for application-level evidence.

## Authority, safety and delivery

- Work on a separate branch for changes. Do not directly push to `dev`, force-push a shared branch, merge, deploy, change production data or expose secrets without explicit task authority and the relevant independent gates.
- Only claim results you personally obtained from commands/tools for the cited commit. Distinguish observed facts, inferences and untested hypotheses.
- If source contradicts these instructions, investigate and propose a correction with evidence; never silently choose whichever version is easier.
- Every PR should explain the problem, ownership, verification, remaining risk and whether review/merge is still pending. Keep historical decisions as historical rather than rewriting them to imply they were always correct.
