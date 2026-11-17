# BDSPro Engineering Charter — Architecture before implementation

**Status:** working research charter, not accepted implementation architecture. **Date:** 2026-11-17.

## North star

BDSPro is expected to grow into a substantial Go business backend. The purpose is not to make current API endpoints work with the fewest lines, and not to clone Gitea. Build a codebase that teams can **understand, extend, upgrade, deploy, monitor, diagnose and maintain** over years while keeping technical complexity controlled.

Production can fail; architecture does not promise zero failure. The objective is to reduce avoidable failure modes, make changes comprehensible, bound blast radius, detect issues early, and make recovery/debugging possible. We do not claim measurable reliability improvement without measurements.

## What counts as a justified architecture decision?

For each contract or pattern, explain:

1. **Problem and system force.** What real engineering constraint exists *independent of today's code*? Example: a resource borrowed by multiple consumers must have a definite lifetime.
2. **Invariant and scope.** What has to remain true, for whom, under which lifecycle phase? Separate startup, serving, shutdown, migrations and deployment.
3. **Mechanism and proof.** Exact source at a pinned commit, entrypoint/caller/callee, state changes, error paths, ownership and interactions with other subsystems.
4. **Alternatives.** At least one viable simpler design and one meaningful different design when applicable. Compare actual complexity rather than aesthetic preference.
5. **Costs.** New code types, hidden dependencies, test isolation, operational requirements, team learning, future change surfaces and failure amplification.
6. **Counterexample.** A case where the preferred design is worse or does not solve the problem.
7. **Acceptance.** Evidence sufficient for an ADR, implementation gates and what production measurements would be required later.

A runtime test can prove a particular behavior under its conditions, **not** that the chosen abstraction is necessary or optimal.

## Evidence and authority

1. Language/runtime/library contracts and standards: Go semantics, net/http, database/transaction/queue specifications, supported official documentation; verify versions.
2. **Actual Gitea source** at a pinned commit and matching callers/consumers, supplemented by Gitea docs/issues/commits when intent is unclear. Gitea is an architectural reference, not an unconditional authority.
3. Other substantial Go systems and engineering literature **only when relevant to a concrete open question**; never invent external support.
4. **BDSPro v4 CRM and current dev** as requirements evidence, architecture experiments and objects of audit. Their existence does not establish correctness.
5. Independent logical analysis, explicit hypotheses, targeted measurements/tests. Do not label derived conclusions as upstream author intent without upstream evidence.

Always label **SOURCE FACT**, **DESIGN INFERENCE**, **HYPOTHESIS**, and **UNVERIFIED**.

## Non-negotiable design discipline

- One clear owner for each responsibility and resource; consumers should not silently assume they own resource cleanup.
- Distinguish package import direction from runtime readiness and object construction from activity activation.
- Errors cross boundaries with semantics intact; failure policies are chosen by the responsible layer. A `log.Fatal` call is not, by itself, a reliability strategy.
- State and concurrency must have explicit lifetimes, cancellation and completion boundaries where needed.
- API/worker side effects, DB transactions, migrations and auth are evaluated for correctness independently of startup convenience.
- Treat observability and operational diagnosis as design considerations, but avoid inventing monitoring infrastructure before defining signals and owners.
- Prefer fewer mechanisms where they satisfy the same proven invariants; do not reward abstraction count or superficial Gitea naming similarity.

## Design economics / measurement

Before production, compare: number of independent owners for a resource, number of touched packages for a representative change, explicit/hidden runtime dependencies, call-graph traceability, number of initialization locations, compile-time enforceable boundaries and test/setup burden. These are *architectural proxies*, not proven reliability gains.

When systems are operational, measure relevant outcomes (e.g., release regressions, error rate, latency, pool saturation, task lag, time to diagnose/restore, resource drain). Never assign invented target percentages.

## Decision discipline

No default `NÊN` for existing `startup.Run`, package global state, Wire, universal `Start/Stop`, or any migration. Refer to [decision protocol](research-protocol.md) and [register](decisions/README.md). Existing BDSPro is **not** an architectural baseline to defend.
