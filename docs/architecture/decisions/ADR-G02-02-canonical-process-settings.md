# ADR-G02-02 — Canonical process settings and directory owners

**Status:** ACCEPTED (explicit user decision: 2026-11-21).
**Owner:** BDSPro backend.
**Reviewed baseline:** `dev@b1ee221c807f5abdbdfd4f3bcd76d17375a04ad7`.
**Implementation status:** pending verification at the final refactor SHA.
**Detailed rationale:** [Canonical Process Settings & Source Ownership](../canonical-settings-and-ownership.md).

## Context / mechanism

BDSPro is presently a single-process Go monolith. PostgreSQL, Redis and
JWT maker are process-owned singletons and most HTTP/service consumers
already read `modules/setting` package globals. After prior refactors,
`WebConfig` was also handed through `cmd → routers → common → resource`,
so two paths to the same values had to be maintained.

The accepted design is **one effective configuration, one validation pipeline,
one process publication**:

- `setting.LoadWebConfig` / `LoadMigrateConfig`: pure typed validation.
- `setting.LoadWebSettings` / `LoadMigrateSettings`: process entrypoint
  publication once, after complete validation.
- `cmd`, `routers`, `models/db`, `modules/conn/auth/graceful`: consume
  the published settings directly when initializing process-owned resources.
- `db.Connect(ctx, source)` and `RedisManager.Client(ctx, address)` remain
  explicit reusable primitives. The pattern is **not universal DI avoidance**.
- `config check`: no publication, log file, network or side effect.

## Evidence

- Gitea shared setting/database: https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/modules/setting/database.go#L19-L54
- Gitea shared DB initializers: https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/models/db/engine_init.go#L48-L77
- Gitea startup composition: https://github.com/go-gitea/gitea/blob/4a9a53d79abc8ab339b18d6c8c59004d7c9d5126/routers/init.go#L114-L145
- Grafana pure config without globals: https://github.com/grafana/grafana/blob/ab4ccc30c29023b0373e646dd6fd8a29348895ab/pkg/setting/setting.go#L1574-L1584
- BDSPro baseline: `cmd/web.go`, `modules/setting/setting.go`,
  `routers/init.go`, `models/db/engine_init.go`, `modules/conn/redis_init.go`.

## Alternatives and rejected extrapolations

1. **Keep multi-hop `WebConfig` passing:** explicit inputs but not end-to-end
   DI; process singletons and HTTP settings globals remain, so the extra
   parameter surface is not justified at current scale.
2. **Create instance-based App/Container:** appropriate if BDSPro needs
   simultaneous application instances or isolated hot reload; not required
   by the current process contract.
3. **Let each subsystem parse its own config:** rejected, duplicates INI/ENV
   policy and risks conflicting source precedence.
4. **Remove all errors/rollback because Gitea uses globals:** rejected;
   config access has no logical bearing on failure cleanup.

## Directory ownership consequences

`cmd` owns CLI process lifecycle; `modules/setting` owns config;
`modules/log` owns writers; `modules/graceful` owns shutdown;
`models/db` owns pool and persistence; `modules/conn` owns Redis;
`routers` owns application route composition and startup ordering;
`services/<domain>` owns business transaction rules; `models/<domain>`
owns entity invariants. `services/context` is a documented transport-facing
Gitea-style exception and will not move without verified benefits.

No synthetic folder, manager, registry or generic helper will be introduced
solely for matching the Gitea tree.

## Guarantees and limits

- One-time publication is guarded; data loaded by the *pure* readers is
  ephemeral and not a parallel long-lived authority.
- Globals must not be written during request handling; Go cannot enforce
  read-only exported values, so this is an engineering contract.
- DB/Redis still need process restart after their Closed singleton lifecycle;
  a hot reload is explicitly outside this ADR.
- Tests must verify failure does not partially publish; config-check pure;
  CLI migration untouched; rollback, graceful, auth and HTTP smoke unchanged.
- Reject new config parser dependencies in resource/domain packages through CI.
- Passing CI does not prove production zero downtime or connection cleanup
  under every failure mode.

## Revisit when

BDSPro adds multiple independently-configured application instances in the
same process, runtime configuration reload, plugin resources with custom
ownership, or measurable engineering difficulties from process globals.
