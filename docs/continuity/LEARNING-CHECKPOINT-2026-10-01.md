# BDSPro Learning Acceptance Checkpoint — 2026-10-01

## Purpose

This checkpoint compresses the verified learning/proof chain reached while evolving
the clean BDSPro Go backend from first principles.

It is not a transcript and not a target architecture. It records:

- what pressures were introduced;
- what mechanisms were learned;
- what source/runtime boundaries were earned;
- what was proven manually or conceptually;
- what remains provisional;
- what the next authorized pressure is.

Authority still follows `MASTER.md`: live source and reproducible proof outrank
this document. Local source may be ahead of remote `main`; reconcile the actual
worktree before source mutation.

## Governing method

The governing method remains Matching Start:

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
  -> DECISION / SOURCE CHANGE
```

The important training loop remains:

```text
UNDERSTAND
  -> PREDICT
  -> TYPE / IMPLEMENT
  -> OBSERVE REAL OUTPUT
  -> EXPLAIN MECHANISM
  -> CONNECT TO ENGINEERING CONSEQUENCE
  -> GENERALIZE
```

The user types/runs code and commands personally. Architecture is derived from
real pressure rather than from a preferred pattern tree.

---

## 1. Minimal Go process and HTTP lifecycle

The backend began with one root `main.go` and stdlib `net/http`.

The first runtime growth established:

- root `main.go` is enough while one executable has no stronger source pressure;
- `PORT` is configuration, not a reason to create a package;
- `http.Server` plus signal handling exposes application/process lifecycle;
- graceful shutdown is not equivalent to immediate process termination;
- an in-flight slow request can finish during graceful shutdown;
- construction, resource acquisition, run, shutdown and release are distinct.

A more explicit listener path was used conceptually:

```text
construct application
construct server
acquire OS listener
run
wait
shutdown
```

Key reusable concept:

```text
RESOURCE:
acquire -> use -> release
```

---

## 2. PostgreSQL became the first required durable dependency

`pgx/v5/pgxpool` was introduced only after the requirement for durable truth.

Important distinctions learned:

```text
pool construction != successful connectivity
Ping              = startup usability proof
```

Application startup was changed conceptually to:

```text
construct pool
  -> Ping with timeout
  -> only then allow HTTP listener
```

A `run() error` boundary became useful because resource-owning code needs normal
return/defer semantics while `main()` owns final process exit policy.

Manual failure proof established:

- database down -> startup denied;
- database up -> database ready, then HTTP listener;
- required startup dependency failure must abort startup.

---

## 3. Docker was earned by local infrastructure lifecycle

PostgreSQL local development was moved behind Docker/Compose while the Go
application remained native in WSL.

The current learning model deliberately separated:

- image;
- container;
- process;
- environment;
- network;
- port publishing;
- writable container layer;
- bind mount;
- named volume;
- Compose model.

PostgreSQL 18 uses the persistent mount target:

```text
/var/lib/postgresql
```

The important lifecycle theorem is:

```text
container lifetime != persistent data lifetime
```

Named volumes exist because database bytes must survive container replacement.

Docker knowledge was then rebuilt from primitives instead of memorized Compose:

```text
docker CLI
  -> Docker Engine
  -> image
  -> container
  -> process
```

### Environment

Learned boundaries:

```text
host shell variable
!=
host exported environment
!=
container configuration
!=
container process environment
```

`.env` has no power by itself. Always ask which program reads it:

- Bash via `source`;
- Docker via `--env-file`;
- Compose via interpolation/environment rules;
- an application via its own loader.

### Networking

Learned:

```text
localhost is context-relative
```

Host-to-container and container-to-container connectivity are different paths:

```text
native host process
  -> host published port
  -> container port

container A
  -> Docker network DNS/service name
  -> container B port
```

Publishing a port does not create a listener; the application must already bind
and listen on the container port.

### Storage

Established:

```text
container writable layer
  -> deleted with container

bind mount
  -> host path owns data lifecycle

named volume
  -> Docker-managed storage owns data lifecycle
```

`docker compose down` and `docker compose down -v` therefore have materially
different data consequences.

### Compose

Compose was understood as declarative orchestration over the same primitives,
not a separate runtime:

```text
compose.yaml
  -> docker compose CLI
  -> Docker Engine
  -> containers / network / volumes
```

Key command distinctions:

- `up`: materialize/reconcile the Compose model and start;
- `start`: start existing stopped containers;
- `stop`: stop processes but preserve containers;
- `down`: remove project containers/network;
- `down -v`: also remove associated volumes;
- `ps`: observe project runtime;
- `logs`: observe stdout/stderr;
- `exec`: start another process inside an existing running container.

---

## 4. Liveness and readiness were separated

Requirement pressure distinguished:

```text
/health = process alive
/ready  = required dependency currently usable
```

Runtime proof:

- stop PostgreSQL while Go process remains alive;
- `/health` continues returning 200;
- `/ready` returns 503;
- restart PostgreSQL without restarting the app;
- readiness recovers.

The important theorem:

```text
alive != ready
```

Startup `Ping` and runtime readiness `Ping` may use the same low-level
mechanism while carrying different semantics.

The first meaningful dependency injection remained simple:

```text
CREATE dependency -> PASS dependency -> USE dependency
```

No DI framework was earned.

---

## 5. Listing introduced the first durable business state

The first durable operation was intentionally small:

```http
POST /listings
{"title":"..."}
```

The first schema established only demonstrated truth:

```sql
CREATE TABLE listings (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title text NOT NULL
);
```

No UUID, audit fields or speculative attributes were added.

The first important durability lesson was:

```text
HTTP 201 != durability proof
```

Durability was independently inspected through PostgreSQL and process restart.

---

## 6. Schema evolution earned migrations

Listing state introduced:

```text
DRAFT
PUBLISHED
```

Schema evolution added status through a new migration rather than rewriting
already-applied migration history.

The migration model learned:

```text
migration
=
historical state transition
```

not merely "the current schema written as SQL".

Data backfill must preserve historical truth rather than invent facts.

The migration runner was earned only after manual SQL replay exposed versioning,
ordering and failure-state pressure.

`golang-migrate` was selected for this pressure.

Important migration concepts:

- ordered versions;
- up/down transitions;
- schema version tracking;
- dirty state;
- locking;
- migration history immutability after application.

A deliberate failing migration proved:

```text
version = N
dirty   = true
```

and that further migration work is blocked until a human inspects/reconciles
actual schema state.

Key theorem:

```text
force != repair database

force
=
declare the reconciled clean version
after human inspection/repair
```

---

## 7. Publish introduced business transition + concurrency

Listing lifecycle became:

```text
DRAFT --Publish--> PUBLISHED
```

Required outcomes:

- valid DRAFT -> success;
- already PUBLISHED -> conflict;
- missing -> not found.

Publish was modeled as an operation:

```http
POST /listings/{id}/publish
```

rather than as a generic status patch.

The first concurrency-safe mechanism was conditional mutation:

```sql
UPDATE listings
SET status = 'PUBLISHED'
WHERE id = $1
  AND status = 'DRAFT'
RETURNING ...
```

This avoided a check-then-act race.

Manual concurrent proof established the invariant:

```text
exactly one request wins DRAFT -> PUBLISHED
```

The second read after a zero-row update only classifies the failure
(not-found vs already-published); correctness comes from the conditional update.

Reusable concurrency reflex:

```text
do not trust an old observation
when correctness depends on current durable state

re-check eligibility at mutation time
```

---

## 8. Application operation boundary and package growth

`Create`, `UpdateDescription` and `Publish` emerged as cohesive Listing
capabilities.

Growth happened through the weakest useful boundary:

```text
inline
 -> function
 -> file
 -> package
```

A first file boundary separated listing behavior from HTTP/process construction
without immediately creating a package.

Once Listing owned enough cohesive vocabulary, state transitions, validation and
application errors, the first package boundary became earned:

```text
main.go
listing/
  listing.go
migrations/
...
```

The package public surface conceptually became:

- `Listing`;
- application errors;
- `Create`;
- `UpdateDescription`;
- `Publish`.

The package still honestly depends on `*pgxpool.Pool`.

A Repository/Store/interface was deliberately not created merely because the
package touches PostgreSQL.

Package boundary bought:

```text
separate namespace
explicit import dependency
explicit public API
compiler-enforced visibility
```

while file boundary had only bought cohesion/navigation.

---

## 9. Business atomicity earned a transaction

Publish later needed two durable facts:

1. `listings.status = PUBLISHED`;
2. one durable publication record.

A `listing_publications` table was introduced.

Existing already-published listings required a truthful backfill. No historical
`published_at` was invented because the historical timestamp was not known.

`Publish` then owned a transaction around:

```text
conditional UPDATE
  +
publication INSERT
  +
COMMIT
```

Important theorem:

```text
SQL statement success != business operation committed
COMMIT               = durability boundary
```

A deliberate constraint failure on the second write proved:

```text
UPDATE executed
  -> INSERT failed
  -> ROLLBACK
  -> listing remained DRAFT
  -> publication row absent
```

This is the strongest reason transaction ownership remains at the business
operation boundary.

Reusable theorem:

```text
transaction boundary tends to follow business atomicity boundary
```

---

## 10. Draft completeness introduced transition eligibility

A Draft may validly exist incomplete while Publish requires completeness.

The first completeness concept used description:

```text
DRAFT
  description may be absent

PUBLISHED
  description must be present
```

This established:

```text
valid to exist != valid to transition
```

Description validation belongs to the Listing capability rather than HTTP
representation parsing.

---

## 11. Published Listing became non-editable in the current model

Current lifecycle pressure states:

```text
DRAFT
  -> UpdateDescription allowed
  -> Publish allowed when complete

PUBLISHED
  -> UpdateDescription rejected
```

Edit eligibility is protected by conditional mutation:

```sql
UPDATE listings
SET description = $2
WHERE id = $1
  AND status = 'DRAFT'
RETURNING ...
```

Manual concurrent proof observed:

```text
PUBLISH HTTP 200
EDIT    HTTP 409
```

That proved one valid interleaving: Publish wins first, then Edit is rejected.

The key concurrency model is not "which request must win"; it is:

```text
which interleavings are allowed?
which committed states are forbidden?
```

Allowed:

- Edit commits first, then Publish;
- Publish commits first, then Edit is rejected.

Forbidden:

- Publish commits first and a later Edit still commits.

---

## 12. Publication Readiness exposed advisory vs authoritative decisions

A new question appeared:

```text
Can the client know why a Listing is not publishable
without actually attempting Publish?
```

This earned a pure policy concept:

```text
publicationReadiness(Listing)
  -> PublicationReadiness
```

It requires no database/network/HTTP mechanism.

The crucial distinction:

```text
READINESS
=
advisory observation of a snapshot

PUBLISH
=
authoritative state transition at mutation/commit time
```

Therefore:

```text
ready=true at T1
does not guarantee
Publish succeeds at T2
```

This is a business-form TOCTOU problem:

```text
TIME OF CHECK != TIME OF USE
```

A read-side pre-check can support UX/explanation but never grants future mutation
rights.

The canonical write path must still re-check current durable truth at mutation
time.

A small amount of policy duplication between:

- pure Go readiness evaluation; and
- authoritative SQL predicates

is currently accepted because replacing it merely to achieve DRY would require
stronger coordination/locking complexity without enough demonstrated value.

Key trade-off:

```text
small duplication
<
unearned concurrency/locking complexity
```

---

## 13. Shell/Bash foundation was rebuilt from first principles

Before automating test workflow, Bash fundamentals were learned because otherwise
`.sh` would become copied incantation.

Core model:

```text
terminal
  -> Bash process
  -> parse command
  -> expansion / argument construction
  -> builtin or executable resolution
  -> child process
  -> exit status
```

Important concepts established:

- Bash itself is a process;
- foreground vs background jobs;
- exit status and `$?`;
- `&&` and `||`;
- stdin/stdout/stderr;
- redirection `>`, `>>`, `<`, `2>`;
- pipes `|`;
- shell variable vs exported environment;
- `source` executes in current shell;
- `PATH` as executable search directories;
- builtin vs external executable;
- quoting with single vs double quotes;
- word splitting and argument boundaries;
- command substitution `$(...)`;
- subshell `(...)`;
- globbing;
- absolute vs relative paths;
- `.`, `..`, `~`, filesystem root;
- WSL path namespace vs Windows path namespace;
- script path vs caller working directory;
- `$0`, `$1`, `$@`, `$#`;
- `dirname`, `basename`, `BASH_SOURCE[0]`;
- `if`, `[[ ... ]]`, functions, `local`;
- `return` vs `exit`;
- `set -e`, `set -u`, `pipefail`.

Cross-platform theorem:

```text
universal concepts:
process
cwd
filesystem path
environment
arguments
exit status
streams
permissions

shell syntax differs:
Bash / zsh / PowerShell / cmd
```

The goal is to transfer the concepts rather than memorize platform-specific
syntax.

---

## 14. Testing foundation was rebuilt before writing integration automation

Testing was derived from claims, not from `testing.T` syntax.

Canonical test reasoning:

```text
REQUIREMENT
  -> INVARIANT
  -> SCENARIO
  -> PRECONDITION
  -> ACTION
  -> OBSERVATION
  -> EXPECTED OUTCOME
```

A test should assert at the level where the invariant lives.

Important distinctions:

```text
unit test
  -> small isolated behavior / pure calculation

integration test
  -> real collaboration/mechanism

failure test
  -> inject failure and verify invariant

concurrency test
  -> assert allowed outcome set / forbidden outcomes

end-to-end
  -> full externally visible path
```

Key theorem:

```text
testable != mockable
```

Do not fake the mechanism the test is meant to prove.

For BDSPro:

- `publicationReadiness` is a natural pure unit-test target;
- transaction/concurrency correctness of `Publish` requires PostgreSQL integration
  evidence.

Go testing mechanics covered:

- `*_test.go`;
- `TestXxx(*testing.T)`;
- `package listing` vs `package listing_test`;
- `t.Error`, `t.Fatal`;
- `t.Run`;
- table-driven tests as a response to repeated test shape;
- `t.Helper`;
- `t.Cleanup`;
- `t.TempDir`;
- `t.Setenv`;
- test isolation;
- test cache and `-count=1`;
- avoid `t.Parallel` until shared-state isolation is understood.

Tests are executable evidence, not mathematical proof that no bugs exist.

---

## 15. Current integration-testing pressure

The current requirement is:

```text
Core Publish invariants must be automatically re-verifiable
against real PostgreSQL,
without any possibility of destroying development database bdspro.
```

The smallest responsible response is:

```text
real PostgreSQL
+
separate database bdspro_test
+
integration build tag
+
explicit database-name safety guard
+
three high-value integration proofs
```

The intended proof matrix:

### Publish success

```text
complete DRAFT
 -> Publish succeeds
 -> status=PUBLISHED
 -> publication row count=1
 -> second Publish=ErrAlreadyPublished
```

### Atomic rollback

```text
complete DRAFT
 -> inject failure on publication INSERT
 -> Publish fails
 -> status remains DRAFT
 -> publication row count=0
```

### Concurrent Publish

```text
two concurrent Publish calls
 -> exactly one success
 -> exactly one ErrAlreadyPublished
 -> exactly one durable publication row
```

The test helper must parse `TEST_DATABASE_URL` and refuse destructive integration
work unless the selected database name is exactly:

```text
bdspro_test
```

Repository/Store/interface is still not earned by this pressure.

---

## 16. Current boundary inventory

Earned:

```text
function boundary
file boundary
listing package boundary
migration lifecycle boundary
database transaction boundary
Docker container/data lifecycle boundary
unit/integration evidence distinction
```

Not yet earned merely from current pressure:

```text
Repository / Store abstraction
persistence interface
internal/
cmd/
service layer
domain layer tree
Makefile
CI
Testcontainers
Dockerfile for the Go application
microservice split
```

---

## 17. Self-study keyword map

These keywords are the most useful anchors for deeper official/documentation
research.

### Linux / Bash / process model

- process vs thread
- parent process / child process
- PID / PPID
- process environment
- exit status
- stdin stdout stderr
- file descriptors 0 1 2
- redirection
- pipeline
- foreground/background job
- Bash builtins
- PATH lookup
- shell parameter expansion
- quoting / word splitting / globbing
- subshell
- command substitution
- current working directory
- absolute path / relative path
- shebang
- executable permission
- Bash positional parameters
- BASH_SOURCE
- set -e / errexit
- nounset
- pipefail

### Docker

- Docker client-server architecture
- Docker Engine / daemon
- OCI image / image layers
- container lifecycle
- PID 1 in containers
- container writable layer
- Docker bridge network
- network namespace
- localhost inside container
- port publishing / NAT
- user-defined bridge network
- embedded DNS / service discovery
- container environment
- bind mount
- named volume
- volume lifecycle
- Docker Compose project
- Compose interpolation
- Compose default network
- docker compose up vs start
- down vs down -v
- docker exec
- docker inspect

### Go runtime / HTTP

- package main
- os.Signal
- signal.NotifyContext
- net.Listener
- http.Server
- graceful shutdown
- context cancellation
- resource ownership
- defer
- main vs run pattern

### PostgreSQL / pgx

- pgxpool
- connection pool construction vs Ping
- PostgreSQL READ COMMITTED
- MVCC
- concurrent UPDATE
- row locking
- statement visibility
- transaction BEGIN / COMMIT / ROLLBACK
- constraint failure
- conditional UPDATE
- RETURNING
- transaction atomicity
- SQLSTATE
- connection lifecycle

### Database schema evolution

- schema migration
- forward/backward migration
- migration immutability
- migration version
- dirty migration
- migration lock
- backfill
- historical truth vs invented data
- expand/contract migration (later, when pressure exists)

### Business-state modeling

- state machine
- transition invariant
- valid state vs valid transition
- command vs generic CRUD update
- optimistic conditional mutation
- check-then-act race
- TOCTOU
- advisory read vs authoritative write
- application error vocabulary
- business atomicity

### Testing

- test case design
- precondition / action / assertion
- Arrange Act Assert
- unit test
- integration test
- end-to-end test
- fault injection
- concurrency testing
- invariant-based testing
- table-driven tests in Go
- subtests
- test fixtures
- test isolation
- deterministic tests
- flaky tests
- t.Cleanup
- build tags
- test caching
- real dependency vs mock
- testable vs mockable

### Architecture reasoning

- cohesion
- coupling
- information hiding
- package boundary
- dependency direction
- weakest sufficient boundary
- YAGNI
- DRY trade-offs
- accidental complexity
- essential complexity
- architecture fitness/evidence
- executable specification
- evolutionary architecture

---

## 18. Next authorized pressure

The next step is not a new Listing feature and not CI.

Current insufficiency:

```text
integration test logic can be automated,
but the environment workflow is still manual:

start PostgreSQL
 -> recreate bdspro_test
 -> apply canonical migrations
 -> inject TEST_DATABASE_URL
 -> run integration suite
```

This workflow is:

- ordered;
- repeated;
- partly destructive;
- easy to mis-target;
- easy to run against stale schema.

Therefore the next smallest response is a project-local shell workflow.

It must be derived from understood Bash/Docker/testing primitives and must start
with safety, not convenience.

First invariant for the next pressure:

```text
A destructive integration-test workflow
must refuse to proceed unless its target database is exactly bdspro_test.
```

Only after the primitive workflow is understood and repeatable should stronger
orchestration boundaries such as Makefile, CI, Testcontainers or dedicated test
infrastructure be reconsidered.
