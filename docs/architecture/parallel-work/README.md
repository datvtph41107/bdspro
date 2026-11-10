# BDSPro × Gitea — Parallel Engineering Workstreams

**Status:** coordination policy / working proposal, 2026-11-10. These documents describe **how separate ChatGPT conversations or coding agents can work on distinct branches**. Opening one chat does not automatically start, monitor, synchronize, or run another chat.

## Outcome and invariant

Move the BDSPro backend toward Gitea's source-level architecture: package and file ownership, imports and dependency direction, function signatures, routing/context lifecycle, domain/service/model transactions, initialization, tests, and operations. **Keep** Go 1.27.3, Gin, SQLC/pgx, PostgreSQL 18, Redis/Asynq, REST and existing Auth/AuthZ/OAuth2 behavior. Do not replace Gin with Chi or SQLC with Gitea's ORM. Gitea is a **source of architectural evidence**, not an instruction to copy business logic.

**Authoritative sources, in order:** current GitHub BDSPro source at exact SHA → current Gitea source at pinned SHA → actual CI logs and tested runtime → reviewed contract in this directory → prior chat narrative. A saved prompt does not override code or a failing check.

As of plan creation, branch `dev` still has generic `modules/web.Router[T]`. P01/P02 changes are on draft [PR #218](https://github.com/datvtph41107/bdspro/pull/218), branch `refactor/gitea-web-ownership-20261110`, **not assumed merged into dev**. Recheck live HEAD before every task.

## Recommended sessions: three producers and one coordinator

| Session | Role | Workstream | Write ownership | Initial mode |
|---|---|---|---|---|
| **Chat 0** | Integrator / reviewer | contracts, PR sequencing, acceptance P07 | orchestration docs, shared CI and cross-package integration only with explicit lease | active now |
| **Chat A** | HTTP/context owner | **P01 + inseparable P02 boundary** | `modules/web/**`; `services/context/**` | implement on existing PR #218 |
| **Chat B** | data/domain owner | **P04 then narrow P05** | `models/**`; only explicitly listed business services | implement from current `dev` in its own branch |
| **Chat C** | runtime/modules owner | **P06** | `cmd/**`; named independent shared modules `setting/log/cache/queue/graceful` | implement from current `dev` in its own branch |
| **Chat D (later)** | API/Auth owner | **P03** | `routers/api/v1/**`, Auth/OAuth route/service adapters | **audit/spec/read-only initially**, code only after foundation gate |
| **P07** | same Chat 0 | final integrated acceptance | test matrix, merger coordination, final report | only when dependent gates meet |

Three **parallel coding producers** are realistic in the first wave (A, B, C). D can review API contracts independently but **must not concurrently rewrite A-owned router call sites**. More chats are not inherently more throughput; the unit of parallelism is **independent write ownership and a frozen interface**.

## Required read order for every newly opened chat

1. This file (shared mission, source of truth).
2. [OWNERSHIP_AND_CONTRACTS.md](./OWNERSHIP_AND_CONTRACTS.md) (permissions and API boundary).
3. [SOURCE_MAP.md](./SOURCE_MAP.md) (Gitea → BDSPro source and ownership mapping).
4. [TRUST_AND_VERIFICATION.md](./TRUST_AND_VERIFICATION.md) (evidence gates and trust ladder).
5. [INTEGRATION_PROTOCOL.md](./INTEGRATION_PROTOCOL.md) (branch/PR/handoff).
6. Exactly one specialist file: [A](./tasks/CHAT_A_HTTP.md), [B](./tasks/CHAT_B_DATA.md), [C](./tasks/CHAT_C_RUNTIME.md), [D](./tasks/CHAT_D_API_AUTH.md), or [0](./tasks/CHAT_0_INTEGRATOR.md).
7. [SKILL.md](./SKILL.md) (execution checklist); compare **live Gitea and BDSPro source** as directed by the selected task.
8. When working on P01, read the existing PR #218 documents at `docs/architecture/gitea-alignment/{README,IMPLEMENTATION_STATUS,CONTINUE_PROMPT,SKILL}.md` **on that PR branch**. They are not necessarily on `dev`.

**Context minimization:** all sessions read a small shared kernel and only their specialist task; do not dump the complete previous 100+ message transcript into every chat. Persist meaningful decisions and test evidence in PR/handoff documents instead.

## P01–P07 mapping and gates

- **P01:** canonical `modules/web` Router, typed handler provider, middleware ordering, pre/post-route semantics and Gin integration — Chat A, blocked by the current P01 CI and duplicate execution paths.
- **P02:** `services/context` request-local Base/APIContext/response/cleanup — **same Chat A** because P01 must call and prove this contract.
- **P03:** API/Auth/AuthZ/OAuth route contracts — Chat D reads/audits now, writes only after HTTP contract handoff and data authority handoff.
- **P04:** `models/db` SQLC/pgx transaction, nested rollback-only and verified session authority — Chat B.
- **P05:** model/domain services naming, ownership and imports — Chat B, scoped by domain and reviewed to avoid changing P03/Auth semantics.
- **P06:** process/module startup, settings, logging, cache/queue/graceful lifecycle — Chat C; `routers/init.go` is reserved for an integration handoff.
- **P07:** final baseline, cross-lane integration, full test/runtime/security/benchmark and merge — Chat 0.

## What success means

Every worker produces **one focused branch, one reviewable PR, reproducible tests, exact before/after links, dependencies and a handoff**. The coordinator never infers "done" from a worker's prose. Staged trust comes from repeatable verification, stable boundaries and evidence of zero hidden behavior changes. Source can be implemented in parallel; merge to `dev` is serialized and conditioned on acceptance, not automatic.

## Pointers

- [source/function mapping](./SOURCE_MAP.md)
- [ownership/contract registry](./OWNERSHIP_AND_CONTRACTS.md)
- [trust and verification](./TRUST_AND_VERIFICATION.md)
- [integration plan and gates](./INTEGRATION_PROTOCOL.md)
- [portable skill](./SKILL.md)
- [handoff template](./HANDOFF_TEMPLATE.md)
- [Gitea handler](https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go), [router](https://github.com/go-gitea/gitea/blob/main/modules/web/router.go), [APIContext](https://github.com/go-gitea/gitea/blob/main/services/context/api.go)

**Planning note:** three sessions do not actually communicate by chat. GitHub PRs, task docs, commit SHAs, CI results, and explicit coordinator decisions are the communication protocol.
