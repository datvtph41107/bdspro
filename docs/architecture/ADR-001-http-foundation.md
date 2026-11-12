# ADR-001: Canonical HTTP foundation

**Status: Proposed — evidence collection only (not Accepted)**

**Date:** 2026-11-12

**Baseline:** BDSPro PR #238, C01/C02 at
`7c18af2fa6c81a9888e970c9b16eb43cb73926c5`.

## Context

BDSPro currently uses Gin route matching with custom `web.Router`,
`routeDef`, `mountDef`, reflective typed action invocation, and a
Gitea-like `APIContext`. CI passed, but success does not demonstrate
that BDSPro should permanently own a second router composition framework.

Gitea's separation of modules, models, services, routers, request context
and application lifecycle is useful; its *mechanism* need not be copied if
modern Go primitives or proven libraries already satisfy the contract.

## Alternatives under examination

| Option | Library router | Handler type | Major trade-off |
| --- | --- | --- | --- |
| A: Chi-native | chi/v5 | http.Handler | One additional dependency; native HTTP group/mount |
| B: Gin-native | gin-gonic/gin | gin.HandlerFunc | Native Gin features; framework-aware route layer |
| C: ServeMux-native | standard net/http | http.Handler | No router dependency; no first-class route group abstraction |
| D: Retain PR #238 | Gin + custom web.Router | typed reflection | Retains current code; BDSPro owns extra routing semantics |

## Contract held constant

Shared request/response schema, strict JSON handling, body limit,
authentication stub, 400/401/404/405/422 statuses, and security header.
The experiment does not validate production actor/session permissions,
SQLC transactions or migration parity; those are separate gates.

## Invariants

- One matcher per HTTP application.
- No domain service/model dependency on HTTP framework.
- Exactly one authoritative request identity and response commitment.
- Authorization before state mutation; service owns transactions.
- No breaking change to existing public `{error, code}` response format.
- Cancelation, cleanup, and graceful shutdown are preserved.
- Dependency lifecycle, security posture, and debug ownership documented.

## Evidence and stop conditions

Evidence: same golden tests; build/race/vet; representative handler diffs;
external package surface; allocations/op, ns/op over repeated runs;
p95/p99 under realistic load before any production migration.

Reject any option that needs a second matcher, duplicates binding/validation,
weakens response failure semantics, or adds framework adapters with no
measurable benefit. Keep existing baseline unchanged until decision.

## Ownership and historical management

Maintainer of record: BDSPro architecture owner.
Implementation evidence: research branch and its CI.
When approved, link the exact tested commit and supersede this ADR with
an Accepted decision specifying migration, rollback, and compatibility.
PR #238 remains Draft until separately integrated or superseded.

## Decision

**Deferred.** No framework has been selected by this ADR.
