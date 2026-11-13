# BDSPro Engineering Agent Guide

This file is the **entrypoint for coding agents and new ChatGPT work sessions** in this repository. It describes work policy; it does not grant autonomous permissions or replace GitHub branch protections.

## Program 01 — Research and engineering inquiry

For research requests about how engineers, communities and companies investigate software development across programming, backend, AI, architecture, product, team practice and operations, first read [Program 01 charter](docs/research/PROGRAM_01.md), [practical reasoning guide](docs/research/PRACTICAL_REASONING.md), [visual evidence guide](docs/research/VISUAL_EVIDENCE.md), [source registry](docs/research/SOURCE_REGISTRY.md) and [discussion backlog](docs/research/DISCUSSION_BACKLOG.md).

**Research discipline:** begin with the practical problem and falsifiable question; locate original sources, contrasting arguments, methods and limitations; trace claims to source code, design discussions, experiments or operational outcomes as applicable. Separate hypothesis, observed evidence, recommendation and verified result. Do not prematurely pick Go projects, frameworks, architecture styles or a prescribed solution. Record the next question and a reproducible, appropriately scoped validation step.

**Scope and authority:** Program 01 is a *research process*, not permission to change production code. Continue to follow the ownership/lease/PR/CI rules below for implementation; never treat outdated branch/PR snapshots or chat Memory as authoritative. The research files are a durable handoff for new chats; ChatGPT Memory is helpful but does not guarantee complete cross-chat recall.

## Task-specific agent skills (proposed)

For a source-level investigation, a scoped backend change, a DB/SQLC migration, or independent PR verification, first read the corresponding workflow in [AI-assisted Engineering System](docs/engineering/AI_AGENT_SYSTEM.md) and `.agents/skills/` (`bdspro-source-investigation`, `bdspro-backend-change`, `bdspro-database-migration`, `bdspro-pr-verification`). Select **only relevant skills**. Skills do not confer tool permissions, update account Memory automatically, replace live source/CI checks or bypass lane ownership and explicit user approvals. Before reading these new skills in an agent workflow, verify that the branch containing them has been reviewed and is intentionally selected; on `dev` these links become canonical only after the stacked PR is merged.

## Before changing code

1. Read [Parallel workstream README](docs/architecture/parallel-work/README.md), [current coordination board](docs/architecture/parallel-work/TASK_BOARD.md), [source mapping](docs/architecture/parallel-work/SOURCE_MAP.md), [ownership and contracts](docs/architecture/parallel-work/OWNERSHIP_AND_CONTRACTS.md), [trust and verification](docs/architecture/parallel-work/TRUST_AND_VERIFICATION.md), [integration protocol](docs/architecture/parallel-work/INTEGRATION_PROTOCOL.md), and [skill](docs/architecture/parallel-work/SKILL.md).
2. Determine your lane **A, B, C, D or 0**, then read **only** its specialist task file in `docs/architecture/parallel-work/tasks/`.
3. Fetch the *actual* latest GitHub `dev`, assigned branch, PR diff and final HEAD CI. Do not assume a prior chat's completion claim is fresh.
4. Read real Gitea source for the corresponding file, its imports and its callers. Compare intent, function design, error propagation and runtime behavior with BDSPro before editing.

## Required architecture

Keep **Go 1.27.3, Gin, SQLC/pgx, PostgreSQL 18, Redis/Asynq and REST**. Match Gitea's code organization and execution principles without replacing Gin by Chi or SQLC by an ORM. No copy-only unused packages, no duplicate HTTP/DB engines, no unwarranted removal of tests.

## Write ownership
- **Lane A:** `modules/web/**` and `services/context/**`; P01/P02; existing PR #218 branch.
- **Lane B:** `models/**` in wave 1; `services/user/**` frozen until Chat 0 reconciles PR #218's current edits and grants a lease; P04/scoped P05; own branch from dev.
- **Lane C:** `cmd/**` and assigned `modules/setting/log/cache/queue/graceful/**`; P06; own independent branch from dev.
- **Lane D:** read-only P03 audit until HTTP/DB gate; afterwards approved API/Auth/OAuth paths on independent branch.
- **Lane 0:** shared contracts, `routers/init.go` wiring, CI/merge and P07 acceptance.

Do not edit code owned by another lane; submit a written cross-owner RFC with exact paths and signatures. Existing P01 PR #218 has historical cross-scope edits that require coordinator reconciliation, not a silent exemption.

## Trust from evidence
Each patch needs commit/PR links, exact Gitea and BDSPro before/after references, focused tests, final HEAD CI conclusion, a failure-case check, boundary audit and a handoff. An in-progress CI or passing earlier commit is not acceptance. Run the advisory `bash tools/parallel/check-owned-paths.sh <lane> origin/dev` after the script is available. It is not itself full semantic verification.

**Do not merge directly into dev**; coordinator verifies and serializes integration. Chats do not coordinate or run in the background automatically; GitHub is the handoff mechanism.

## Starting a new session
Copy the `START NEW CHAT` prompt at the bottom of the selected [specialist task](docs/architecture/parallel-work/README.md). If these files are not yet merged into dev, read them on branch `docs/gitea-parallel-harness-20261110`.
