# Current Checkpoint

Updated: 2026-10-01

## Source identity

Active repository:

`datvtph41107/bdspro`

Branch:

`main`

Operational Go module identity for the current phase:

`github.com/datvtph41107/bdspro`

Preferred local workspace:

`~/projects/bdspro`

Important authority caveat: the user's local source/proof may be ahead of remote
`main`. Before any source mutation, inspect the actual worktree and reconcile it
with live Git. Do not overwrite unexplained local work merely to match this
document.

## Current phase

```text
B1 — Matching Start / Source Growth From First Principles
STATUS: ACTIVE
```

The detailed acceptance report for the current learning milestone is:

`docs/continuity/LEARNING-CHECKPOINT-2026-10-01.md`

## Proven/established progression

The current learning/proof chain has advanced through:

```text
minimal Go HTTP process
 -> configurable port
 -> graceful shutdown
 -> explicit listener/resource lifecycle
 -> PostgreSQL required startup dependency
 -> pgxpool + Ping
 -> Dockerized local PostgreSQL
 -> liveness vs readiness
 -> first durable Listing state
 -> schema migrations
 -> migration version/dirty-state recovery reasoning
 -> Listing DRAFT/PUBLISHED state transition
 -> concurrency-safe conditional mutation
 -> application operation boundary
 -> transaction across multiple durable facts
 -> rollback failure proof
 -> file boundary
 -> listing package boundary
 -> Draft completeness / Publish eligibility
 -> non-editable Published state
 -> concurrent Edit vs Publish proof
 -> Publication Readiness
 -> advisory read vs authoritative mutation / TOCTOU
 -> Bash/Shell foundation
 -> Docker image/container/process/network/storage/Compose foundation
 -> testing foundation
 -> current real-PostgreSQL integration-evidence pressure
```

## Current important theorems

```text
process alive != application ready

pool construction != successful connectivity

HTTP success != durability proof

valid to exist != valid to transition

statement success != business operation committed

file cohesion != package dependency boundary

readiness observation != future mutation right

TIME OF CHECK != TIME OF USE

testable != mockable

transaction/concurrency semantics must be tested
with the mechanism that actually provides them
```

## Current requirement

```text
Core Publish invariants must become repeatable executable evidence
against real PostgreSQL.

Integration testing must use a disposable database named bdspro_test
and must refuse destructive work against the development database bdspro.
```

High-value integration evidence:

1. successful Publish creates both durable facts and a second Publish is rejected;
2. failure of the publication write rolls back the Listing status change;
3. two concurrent Publish calls produce exactly one winner and one durable
   publication record.

## Current testing boundary

Pure policy such as `publicationReadiness(Listing)` belongs naturally in fast
unit tests.

Publish transaction/concurrency correctness depends on real PostgreSQL semantics,
so the current evidence must use real PostgreSQL rather than a fake repository.

A Repository/Store/interface is therefore still not earned by testing pressure.

## Current environment workflow

The integration proof currently requires an ordered workflow:

```text
start PostgreSQL
 -> create/reset bdspro_test
 -> apply canonical migrations
 -> export TEST_DATABASE_URL
 -> run go test -tags=integration -count=1
```

This workflow is still manual.

## Pressure now authorized

The next demonstrated pressure is repeatable and safe orchestration of the
integration-test environment.

The workflow is:

- repeated;
- ordered;
- partly destructive;
- sensitive to the selected database;
- sensitive to migration/schema freshness.

The smallest responsible response is one project-local Bash workflow.

The first invariant of that workflow is:

```text
never perform destructive integration-test setup
unless the target database is exactly bdspro_test
```

Build the script from understood primitives rather than hiding them.

## Next proof chain

```text
known project root
 -> required local config resolved
 -> PostgreSQL started/usable
 -> target DB identity proven safe
 -> bdspro_test recreated
 -> canonical migrations applied from zero
 -> integration suite executed with -count=1
 -> run the workflow twice successfully
 -> deliberately dirty bdspro_test and prove the workflow restores known state
 -> prove development database bdspro is unchanged
```

## Not earned yet

Do not introduce merely because integration tests exist:

- Repository/Store abstraction;
- persistence interface;
- `internal/`;
- `cmd/`;
- service/domain/repository layer tree;
- Makefile;
- CI;
- Testcontainers;
- Dockerfile for the Go application;
- application containerization;
- microservice split.

Each requires its own demonstrated pressure.

## Recovery note

Before continuing source changes, recover the live worktree and verify which of
the described local experiments/tests are actually materialized in source.
Conversation proof is useful orientation, but live source and reproducible test
output remain the authority.
