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

The response is not preselected as one Bash script. The current pressure is to
create clear, independently executable project capabilities and then let observed
control-flow/safety pressure decide whether each capability belongs directly in a
Make recipe, a focused Bash script, or later composition.

The first invariant of that workflow is:

```text
never perform destructive integration-test setup
unless the target database is exactly bdspro_test
```

Build every capability from understood primitives rather than hiding them.
Makefile is now a legitimate candidate because recurring project operations have
appeared, but it must be earned target-by-target rather than introduced as a
finished task framework.

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
- CI;
- Testcontainers;
- Dockerfile for the Go application;
- application containerization;
- microservice split.

Each requires its own demonstrated pressure.

## Developer-immersion objective — 2026-10-01

BDSPro is now explicitly a long-running professional practice system, not only a
feature implementation project.

Mandatory recovery document:

`docs/continuity/DEVELOPER-IMMERSION.md`

Recurring work should deliberately exercise real developer surfaces: Git state
inspection, Bash/process reasoning, Docker and PostgreSQL operation, primary
documentation/source tracing, hand-written Go/test code, failure diagnosis, diff
review and coherent commits.

Difficulty should grow from happy paths toward configuration mistakes, stale
state, partial failure, concurrency and recovery only when each new pressure has
clear engineering or learning value.

Current local/remote reconciliation evidence supplied by the user:

```text
local HEAD:        1ce48eda81e0a22ca16d3cb2c89eee6fb53ba254
local origin/main: bf5e7d266633b8cdc806e1b8dffc7814f74355e0
remote GitHub:     advanced beyond local origin/main
```

Therefore no pull/rebase/reset or new source mutation is authorized until
`git fetch origin` refreshes the remote-tracking ref and the relationship is
classified from Git evidence.

After reconciliation, Makefile is now a legitimate *candidate boundary* because
recurring project operations and a need for independently executable project
capabilities have appeared. It must still be introduced incrementally and typed
by the user, beginning with a non-destructive capability such as `postgres-up`.
Bash should be earned only where imperative/safety logic exceeds a simple Make
recipe.

## Local Git reconciliation — observed 2026-10-01

The user ran the recovery inspection locally and produced real Git evidence:

```text
before fetch:
  HEAD        = 1ce48eda81e0a22ca16d3cb2c89eee6fb53ba254
  origin/main = bf5e7d266633b8cdc806e1b8dffc7814f74355e0
  status      = main...origin/main [behind 8]

working tree:
  modified:   main.go
  untracked:  .gitignore, .vscode/, compose.yaml, go.mod, go.sum,
              listing/, migrations/, test-integration.sh

after git fetch origin:
  origin/main = b47e4195581929e26f55a2942d3c2b7dfa5e2c6c
  rev-list --left-right --count HEAD...origin/main = 0 25
```

Interpretation:

- the pre-fetch `origin/main` was a stale local remote-tracking ref;
- fetch updated remote knowledge without changing the working tree;
- local committed history has no local-only commits relative to current
  `origin/main`;
- local `HEAD` is 25 commits behind the current remote main;
- substantial implementation work exists only in the working tree/untracked
  files and must be preserved and understood before synchronization.

This is now the immediate pressure. Do not pull, reset, clean, rebase or overwrite
those files. First inventory the local diff/source, identify coherent work units,
then choose a branch and commit boundaries that tell the true work history.

## Recovery note

Before continuing source changes, recover the live worktree and verify which of
the described local experiments/tests are actually materialized in source.
Conversation proof is useful orientation, but live source and reproducible test
output remain the authority.

## Active coding gate — 2026-10-01

The working method has been strengthened. Before the next source/tooling
implementation, recover and follow:

`docs/continuity/PRACTICE-PROTOCOL.md`

The user must write the code personally. The assistant should expose the
requirement, pressure, vocabulary, relevant API/docs and the smallest syntax
shape needed, then review the user's actual implementation and real output.

For unfamiliar library/tool mechanisms, primary documentation/source tracing and
an explanation in the user's own words are part of the acceptance evidence.

Current immediate source position:

```text
NO new Makefile/task source should be assumed to exist from conversation alone.
First reconcile the local worktree.
Then, if no conflicting local work exists, the next tiny implementation exercise
is to derive and hand-write the first non-destructive project capability:
postgres-up.
```

The purpose is not to finish tooling quickly. It is to build the developer
reflex: requirement -> mechanism -> docs/source -> code -> run -> diagnose ->
explain.

## Daily workflow standard — 2026-10-01

A concrete operating standard now exists at:

`docs/continuity/ENGINEERING-WORKFLOW.md`

For new or materially changed commands/APIs, work must expose: why now, command/API
shape, material arguments, state read/mutated, primary source, prediction, user-typed
execution, observed output and justified conclusion. Familiar repeated commands are
then intentionally compressed to build speed and reflex.

This standard applies across Git, Bash, Docker, PostgreSQL/SQL and Go.

## Contextual-rigor gate — 2026-10-01

Before the next implementation/CLI step, do not automatically select the most
verbose command, strongest guard, deepest abstraction or most production-like
workflow. First identify the current objective, time pressure, failure cost,
expected reuse/lifetime and blast radius.

For every new material flag or option, explain the default behavior, what the
flag changes, why that change is useful here, and what would happen if it were
omitted.

This applies immediately to the current Git recovery work as well as later
Docker, PostgreSQL, Make/Bash and Go implementation decisions.

## Local recovery branch established — 2026-10-01

The user's local accumulated implementation has been moved off local `main`
without losing working-tree changes.

Observed local state:

```text
branch: recovery/local-implementation

modified:
  main.go

untracked:
  .gitignore
  .vscode/
  compose.yaml
  go.mod
  go.sum
  listing/
  migrations/
  test-integration.sh
```

This proves the branch switch preserved the working tree.

Immediate pressure: preserve the accumulated implementation honestly in Git
without fabricating a false fine-grained historical sequence. The next decision
is how to create a truthful recovery checkpoint, exclude local-only artifacts,
and then synchronize onto current remote main before normal incremental work
resumes.

## Recovery implementation staged — 2026-10-01

Local branch:

`recovery/local-implementation`

The accumulated implementation checkpoint has been staged into the Git index.

Staged:

- `.gitignore`
- `compose.yaml`
- `go.mod` / `go.sum`
- modified `main.go`
- `listing/listing.go`
- unit and integration tests under `listing/`
- migrations `000001` through `000004`, up/down
- `test-integration.sh`

`.vscode/` remains outside the staged snapshot.

Immediate next proof: review the exact index snapshot relative to `HEAD` before
creating the recovery commit. Do not fabricate a historical sequence of past
milestones; preserve the accumulated implementation truthfully as one recovery
checkpoint, then resume normal incremental Git discipline.
