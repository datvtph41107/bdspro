
> **2026-11-25 governance reset:** This content is a historical snapshot/reference, not an active mandate or verified current runtime. Recheck live source and requirements. Do not resume Gitea/F40/parallel plans or infer authorization from it. Current policy: https://github.com/datvtph41107/bdspro/blob/dev/AGENTS.md (on the reset branch, use its local root AGENTS.md).

# G02 — Application Runtime Architecture: research agenda

**Status: OPEN / ARCHITECTURE RESEARCH.** Do not conflate current implementation with accepted design.
**Scope:** application/process entrypoints; initialization and dependency topology; resource ownership; long-running runtime activities; error and diagnostic boundaries; extensibility to CRM-sized business scope.

## The actual design problem

The question is NOT whether `mustInit`, `startup.Run`, `InitialApp`, Wire or one manager is good. The question is how a substantial Go monolith can maintain coherent application construction, shared-resource ownership, explicit dependency guarantees, serving and worker lifecycle, and intelligible diagnostics **while business modules and delivery modes grow**.

Gitea is a rich source of actual mechanisms, not a template to copy blindly. Old BDSPro CRM v4 demonstrates domain/dependency scale; current BDSPro dev is under architectural audit, not a proven target.

## Study through concrete design forces

### G02-01 — Composition and change surface
- Trace Gitea `cmd/runWeb → serveInstalled → routers.InitWebInstalled → NormalRoutes → listen` with exact calls; separate process work from router responsibilities.
- Trace CRM v4 `main → cmd/grpc → wire.InitializeApp → InitialApp → grpc.Register* → process.Run`.
- Question: when adding one feature with no new runtime resource, what must change? When adding one *new subsystem/resource*, what must change? Count affected files, imports and construction paths.
- Compare explicit package init, manual wiring and compile-time DI without selecting a winner in advance.

### G02-02 — Temporal dependency and readiness
- Distinguish compiled dependency import, constructor dependency, and temporal requirement (“B ready before A begins”).
- Examine Gitea settings, DB/migration, model/service, gitproxy before mirror services and serving boundary.
- Compare hardcoded ordering, named steps, dependency declarations and state-carrying constructors. Identify invariants each can and cannot prove.

### G02-03 — Ownership, sharing and shutdown
- Trace who creates DB/cache/Redis/queue and who borrows them; examine whether shutdown depends on registered activities completing.
- Distinguish init failure cleanup, process termination, and worker task-draining responsibilities.
- Explain the limits of global package state (test isolation, multiple instances) and explicit construction (wiring/change cost).

### G02-04 — Background and process-mode composition
- Trace Gitea process/queue/cron registration and a consumer's full lifecycle.
- Compare CRM v4 gRPC scheduler and independent payment event consumer, without assuming BDSPro must adopt their microservice topology.
- Ask where Web, Worker, CLI migrations and integration consumers share code vs resources; record real evidence.

### G02-05 — Error, diagnosis and operational contracts
- Distinguish signal propagation, retry policy, fatal boundary, resource closure, logging and supervisor restart. Do NOT elevate fail-fast into a universal goal.
- Analyze Gitea DB and cache retry vs queue nil/error behavior using exact callers.
- Identify diagnostic questions: what failed, at what stage, who owns it, what change triggered it, which request/process is affected; what can static source prove versus what needs live operational data?

### G02-06 — Evolution cost and falsification
- Use representative domain expansion (e.g., CRM-sized) and a non-domain infrastructure change.
- Compare changed package count, implicit assumptions, constructor parameter growth, cross-domain coupling, workflow onboarding and test setup.
- Find a counterexample for each preferred model. Avoid invented reliability gains or “70%” claims without evidence.

## Output required before implementation

For each G02-xx: source-based call graph, underlying engineering constraint, explicit invariants, alternative code shapes, measured/static change costs where possible, counterexample, BDSPro architectural gap and **OPEN/PROPOSED/ACCEPTED** status.

**Current decision:** none of `startup.Run`, global runtime state, Wire/DI, rollback framework or universal lifecycle interface is pre-approved. Evaluation may lead to retaining, replacing or eliminating existing BDSPro mechanisms.

## Research checkpoint / next task

Begin with **G02-01**, not a code refactor. Produce one side-by-side entrypoint and composition trace from pinned sources, then show two code shape options and their effect on adding a CRM-sized subsystem. After that investigate resource/state contracts in G02-02/03.

Sources and refresh rules: [source evidence map](source-evidence-map.md); [research protocol](research-protocol.md); [decision register](decisions/README.md).
