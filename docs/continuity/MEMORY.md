# Durable Memory

## Active identity

Active repository:

`datvtph41107/bdspro`

Preferred WSL workspace:

`~/projects/bdspro`

Operational module path for the current phase:

`github.com/datvtph41107/bdspro`

Product identity remains BDSPro; source-host ownership may evolve later if a real
publishing/ownership/collaboration pressure appears.

## Learning objective

The user is building BDSPro as a clean-slate Go backend in order to learn backend
engineering mechanisms deeply rather than memorize patterns, commands or a final
architecture.

The user types/runs implementation steps personally.

Default learning loop:

```text
UNDERSTAND
 -> PREDICT
 -> TYPE / IMPLEMENT
 -> OBSERVE REAL OUTPUT
 -> EXPLAIN MECHANISM
 -> CONNECT TO ENGINEERING CONSEQUENCE
 -> GENERALIZE
```

## Governing architecture method

Durable method authority:

`docs/continuity/MINDSET.md`

Single fresh-context bootstrap:

`docs/continuity/CONTINUATION-PROMPT.md`

Core chain:

```text
CURRENT REALITY
 -> INTENT
 -> NEW REQUIREMENT
 -> PRESSURE
 -> INSUFFICIENCY
 -> SMALLEST RESPONSIBLE RESPONSE
 -> EARNED BOUNDARY
 -> MECHANISM
 -> EVIDENCE
 -> TRADE-OFF
 -> DECISION
```

Prefer the weakest boundary that solves demonstrated pressure.

Do not create conventional Go/Clean/DDD directories or interfaces merely because
mature repositories contain them.

## Current acceptance artifact

The comprehensive accepted learning checkpoint through 2026-10-01 is:

`docs/continuity/LEARNING-CHECKPOINT-2026-10-01.md`

Read it when deeper recovery than `CHECKPOINT.md` is required.

## Core mental models already established

### Runtime / resource lifecycle

```text
construct
 -> acquire
 -> prove usable
 -> run
 -> wait
 -> shutdown
 -> release
```

### Liveness / readiness

```text
alive != ready
```

### Durable truth

```text
HTTP success != durability proof
statement success != transaction commit
```

### Business state

```text
valid to exist != valid to transition
```

### Concurrency

```text
avoid check-then-act when correctness depends on current durable state

conditional mutation:
WHERE current_state_is_still_eligible
```

Allowed interleavings may vary; forbidden committed states must not.

### Atomicity

```text
transaction boundary tends to follow business atomicity boundary
```

### Read vs write authority

```text
readiness / pre-check
=
advisory snapshot

authoritative mutation
=
re-check current durable truth at mutation/commit time
```

```text
TIME OF CHECK != TIME OF USE
```

### Source boundaries

```text
file boundary
-> cohesion/navigation

package boundary
-> namespace + imports + API + compiler visibility
```

### Testing

```text
Requirement
 -> Invariant
 -> Scenario
 -> Precondition
 -> Action
 -> Observation
 -> Expected outcome
```

```text
testable != mockable
```

Do not fake the mechanism that provides the invariant being tested.

### Shell/process model

```text
Bash parses command
 -> performs expansion
 -> builds argv
 -> resolves builtin/executable
 -> starts process
 -> observes exit status
```

Universal concepts transfer across Linux/macOS/Windows even when shell syntax
changes: process, cwd, paths, environment, argv, streams, exit status,
permissions.

### Docker model

```text
docker CLI
 -> Docker Engine
 -> image
 -> container
 -> process
```

```text
host environment != container environment
host filesystem  != container filesystem
host localhost   != another container's localhost
```

Persistent database bytes need storage whose lifetime is independent of a
container object.

## Current earned source/system boundaries

Earned through demonstrated pressure:

- root single executable;
- function boundaries;
- Listing file/package boundary;
- PostgreSQL required dependency lifecycle;
- migration lifecycle/history;
- transaction boundary for Publish;
- Docker container/network/volume lifecycle for local PostgreSQL;
- unit vs integration evidence boundary.

Still unearned from current evidence:

- Repository/Store interface;
- domain/service/repository layer tree;
- `internal/`;
- `cmd/`;
- Makefile;
- CI;
- Testcontainers;
- application Dockerfile/containerization;
- microservice topology.

## Current target

Current requirement:

```text
Make core Publish invariants repeatably testable
against real PostgreSQL using disposable bdspro_test,
without risking development database bdspro.
```

Current next pressure:

```text
integration-test logic can be automated,
but environment setup is still a repeated,
ordered, destructive manual workflow.
```

Next smallest response:

```text
one project-local Bash integration-test workflow
with database-identity safety first
```

No Makefile, CI or Testcontainers before that primitive workflow is understood
and proven.

## Deep-study keyword anchors

For deeper self-study, use the keyword map in
`docs/continuity/LEARNING-CHECKPOINT-2026-10-01.md`.

Highest-value clusters:

- Bash process model, argv, environment, file descriptors, redirection, PATH,
  parameter expansion, BASH_SOURCE, errexit/nounset/pipefail;
- Docker client-server architecture, image layers, container lifecycle, network
  namespace, bridge networking, port publishing, embedded DNS, bind mounts,
  named volumes, Compose lifecycle;
- Go `net/http`, `context`, signals, resource ownership, testing package,
  build tags, table-driven/subtests, cleanup;
- pgxpool, PostgreSQL MVCC, READ COMMITTED, concurrent UPDATE, row locks,
  conditional mutation, transactions, constraints;
- schema migration versioning, dirty state, backfill, migration immutability;
- state machines, invariants, TOCTOU, advisory read vs authoritative write,
  business atomicity;
- unit vs integration vs failure vs concurrency testing, fault injection,
  deterministic evidence, flaky tests;
- cohesion, coupling, information hiding, evolutionary architecture, weakest
  sufficient boundary, YAGNI/DRY trade-offs.

## Recovery safety

Live source and completed reproducible proof outrank this memory.

Local source may be ahead of remote `main`. Never reset, clean, rebase, amend or
overwrite unexplained local work during recovery.

Conversation memory is orientation only; durable docs plus live source/proof are
the continuity bridge.
