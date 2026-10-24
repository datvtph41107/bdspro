# ADR-0001 — Own shared capabilities without rebuilding BDSPro as a framework

- **Decision:** Accepted when ENG-GOV-01 merges into `dev`
- **Date:** 2026-10-24
- **Scope:** Engineering governance only; no behavior, DB schema or Business Core change

## Context

BDSPro's active runtime uses Go/Gin, PostgreSQL/sqlc, Redis/Asynq, HTTP handlers, `auth`, `token`, `worker` and `main.go`. The source already has working request admission, password/JWT sessions, organization SQL membership authorization, Google OIDC with configured credentials and development-only Phone OTP.

Two useful references have different strengths:

- DRX Laravel exposes `IPActive`, `PreLogin`, `AuthenticateUser` and `SufLogin` in route composition. In its existing code, `PreLogin` also verifies credentials, while `SufLogin` may change security/device/token state after downstream processing. BDSPro must preserve behavior-based names and not inherit those ambiguous boundaries.
- Java DRWash separates Gradle `core` from `api`, `admin`, `store` and `IoT`. The shared `OrderCoreService` illustrates common domain ownership, but one large Core service can accumulate unrelated responsibilities. BDSPro has no demonstrated need for multiple Go modules or applications yet.
- TechSchool SimpleBank helps keep Go and SQL execution paths direct and reviewable.

The missing part was an explicit, team-readable contract for capability ownership, durable invariants and change acceptance. Adding architectural layers before business requirements are known would increase maintenance with no proven benefit.

## Decision

1. **Adopt ownership by behavior and state**, not by a universal layer template. Maintain [Core Ownership Matrix](../core-ownership.md) and [Development Contract](../development-contract.md) against actual source.
2. **Keep a single Go application/module** for current Gin HTTP + worker execution. No artificial `core`, `service`, `repository`, `adapter` hierarchy is mandated.
3. Preserve existing explicit responsibilities: `api` handles HTTP/request policy, `auth` and `token` handle identity/session mechanisms, PostgreSQL SQL/constraints enforce persisted invariants and resource ownership, `worker`/`mail` handle async processing.
4. **Prefer DB authorization predicates on mutations** when permission must be checked atomically with state change, rather than client roles or JWT-only owner flags.
5. Create abstractions/interfaces only with a demonstrated shared use case, replaceable implementation, testing seam or real responsibility boundary. Existing interfaces are not removed merely to satisfy this guideline.
6. Use a concise PR template. Do **not** invent personal ownership or add `CODEOWNERS` until concrete maintainers and review responsibilities are agreed.
7. A passing CI run verifies the tested scope, **not** the user's WSL state, durable recovery, real provider delivery or production hardening.

## Consequences and limits

**Benefits:** newcomers can locate the behavior to change, reviewers can trace invariants, and future business modules can consume common identity and organization capabilities without copying the authentication system.

**Costs:** the ownership matrix must be maintained when source changes. A current simple operation may legitimately call a Store from an HTTP handler; no forced service layer means discipline comes from review and tests.

**Not resolved by this ADR:** G0 WSL acceptance, PostgreSQL–Redis atomic handoff, email-verified account policy, real Google/SMS production E2E, full Business Core modeling.

## Reconsider when

Revisit only after at least one concrete pressure appears: two independent entrypoints need the same business rules, a package has conflicting owners, repeated code creates observable defects, or deployment/runtime isolation provides measurable benefit. Record the change, its tradeoffs and the migration path before introducing extra modules.

## Alternatives deliberately not chosen

- **Copy Laravel DRX middleware semantics:** mixes credential checks and post-response token/OTP changes in ways not suited to BDSPro's current trust boundaries.
- **Copy Java DRWash's multi-module Core:** adds packaging/integration costs without a second BDSPro application.
- **Introduce a universal Clean Architecture skeleton:** forces interfaces and service layers before they own a real invariant.
- **Leave ownership implicit:** makes onboarding/review inconsistent and permits unauthorized duplication of identity/session logic.
