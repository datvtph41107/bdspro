---
name: verify-bdspro
description: Verify BDSPro first-party sessions and organizations through real Gin routes and disposable PostgreSQL/Redis, capturing a repeatable evidence record. Use after changing auth/session/org behavior or before claiming such changes are verified.
disable-model-invocation: true
---

# Verify BDSPro (ENG-AI-02)

This is the *current* verification control skill, not an AI's personal checklist. It uses existing production Gin registration and acceptance test helpers, real PostgreSQL 18 and Redis 7, while HTTP requests run through Go's in-process `httptest` adapter. It does **not** launch the production HTTP listener or test network ingress.

## Launch

From repository root, with Docker and Go **1.27.3** installed:

```sh
bash .cursor/skills/verify-bdspro/scripts/verify.sh
```

The runner refuses a missing Docker daemon or wrong Go version. It creates two uniquely named, unmounted Docker containers with loopback-only ephemeral ports and PostgreSQL database `bdspro_test`, applies the repository's canonical SQL migrations, and sets `BDS_TEST_MODE=disposable` for the existing HTTP acceptance suite. It does not read or modify a developer's app.ini or local database.

## Doctor

Before any drive, the runner checks PostgreSQL readiness, Redis `PONG`, and executes a *read-only* `select current_database()` that must return exactly `bdspro_test`. An unhealthy service or incorrect DB identity is **BLOCKED**, never a pass. Tests themselves also fail if their database name is not `bdspro_test` or disposable mode is absent.

## Drive

The runner executes the following *existing* tests with `-count=1`:

- [Live session revoke](../../../../routers/g1_session_authority_e2e_test.go): register/login over Gin, revoke, inspect persisted blocked session, verify rejected access and refresh.
- [Organization access](../../../../routers/organization_access_e2e_test.go): HTTP create/member authorization, read projections, denial and post-mutation visibility.
- [Team invitation](../../../../routers/team_invite_e2e_test.go): authorized team creation, invitation/outbox persistence, denied actor, single-use redemption and replay rejection.

See [session feature](features/sessions.md) and [organization feature](features/organizations.md) for request/SQL/verification anchors. A Go test returning 0 is **not** sufficient: the runner checks that each named test actually ran and reported `PASS`. A skipped or unexecuted case fails.

## Evidence

The runner writes `data/verification/<run-id>/` (ignored by Git): `manifest.txt` with commit SHA, `migrate.log`, `drive.log`, and `verdict.txt` only if all assertions pass. The directory is mode 700; logs can contain sensitive test diagnostics and must not be uploaded without review. GitHub Actions independently records the step result. A PASS covers *only* the three named capabilities on the cited commit and in-process Gin transport; claims about production providers, full network hosting, concurrency exhaustiveness and latency require different evidence.

## Cleanup

The script installs an EXIT trap and removes only containers it created, by their unique per-run names. It keeps evidence on disk. It does not execute `docker compose down`, prune Docker, stop developer services or delete application data. If the process is killed ungracefully (e.g. SIGKILL), inspect `docker ps --filter label=bdspro.verify` and clean only the specific orphaned verification containers after confirming ownership.

## Extend and maintain

A new feature may be added only after identifying the actual route and owner, choosing a stable acceptance test with real observable side effects, and adding its exact name to the runner's checked list. Test a negative case: remove or misspell one selected test name and confirm the runner fails rather than treating a no-tests-run result as success. Keep feature maps in sync when routes or DB owners change. Source evidence supersedes the map; repair drift in a separate, focused PR.
