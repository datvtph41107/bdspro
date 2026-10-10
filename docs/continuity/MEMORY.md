# Durable Memory

> **2026-10-10 ACTIVE BUSINESS→CORE MEMORY (durable, supersedes stale 2026-10-06 rollout status):** Business Atlas across ALL BDSPro domains is at scenario/one-row/hazard/GATE coverage level, not field-verified. Canonical DBML V0.4 has **164 candidate tables** (old 160 + source_intakes, source_price_reports, crm_inquiries, crm_followup_tasks), six added same-tenant refs; NO runtime migrations / SQL proof. First read [2026-10-10 Business-to-Core checkpoint](checkpoints/2026-10-10-business-to-core/README.md), [Business Atlas V1](../business/BUSINESS-ATLAS-V1-2026-10-10.md), and [code gates](checkpoints/2026-10-10-business-to-core/BUSINESS-TO-CORE-IMPLEMENTATION.md). "Tiếp tục" → resume from most recent proved gate, currently **ENG-00 inspect dev/local state → CORE-01 C01 exact-one Party subtype lab**; do not restart BUSINESS-03. Product BUSINESS-00 adoption/WTP gate remains open. Actor≠Account≠Membership≠Role≠right, source intake≠Property≠Listing; Inquiry≠Contact≠Opportunity; Task≠Activity; subscription≠legal permission; group≠tenant CRM access. The user wants to move to **hands-on code** one pressure/test at a time, usually typing SQL/Go personally. Git files are reproducible recovery authority, conversational Memory is not a guaranteed complete transcript.

---


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
reconcile local Git state
-> expose recurring project operations as small named capabilities
-> let pressure decide whether each belongs in Make or focused Bash
-> compose only after the primitives are understood
```

Makefile is now a legitimate candidate because recurring project operations and
independent execution/debugging needs have appeared. It must be earned target by
target, not introduced as a finished framework. CI and Testcontainers remain
later decisions.

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

## Active coding and source-trace invariant — 2026-10-01

The user explicitly changed the training contract to optimize for long-term
engineering fluency rather than fast artifact production.

Canonical protocol:

`docs/continuity/PRACTICE-PROTOCOL.md`

Remember:

```text
Do not normally give a complete function/test/file before a real user attempt.
Require the user to derive names/responsibilities, write the code, run it, read
real output and explain the mechanism.
```

When an unfamiliar library function/tool behavior matters, help locate primary
documentation and source, then require an own-words interpretation before the
mechanism is treated as learned.

Use SimpleBank and other OSS as cases to trace and reproduce independently, not
as code to copy.

Slower progress is intentionally accepted because repetition, debugging and
source tracing are expected to build practical reflexes and career-level
transferable skill.

## Developer immersion invariant — 2026-10-01

BDSPro is the user's deliberate-practice environment for becoming operationally
fluent as a backend developer. The goal is repeated real use of Bash/processes,
Git, Docker, PostgreSQL/SQL/CLI, Go, testing, documentation/source tracing and
debugging until the user can operate them naturally and transfer the reasoning to
future projects.

Canonical long-horizon model:

`docs/continuity/DEVELOPER-IMMERSION.md`

AI should increasingly act as a research/review/reasoning partner rather than an
implementation generator. Experience evidence should come from work the user
actually typed, operated, failed, debugged and can explain.

## Current local Git recovery fact — 2026-10-01

After `git fetch origin`, the user's local committed history is 0 commits ahead
and 25 commits behind `origin/main`. The working tree contains substantial
uncommitted implementation work: modified `main.go` plus untracked Go module,
Compose, Listing, migrations and integration-workflow files.

The immediate task is therefore not new feature/tooling code. It is to inspect,
understand and preserve that local work, then establish truthful branch and commit
boundaries before synchronizing with remote main.

## Daily command reflex — 2026-10-01

Canonical standard: `docs/continuity/ENGINEERING-WORKFLOW.md`.

First encounter/new option: trace problem -> state -> syntax -> primary docs -> prediction
-> user types -> real output -> explanation. Repeated familiar use should become concise
and fast. Expand again when flags, environment, failure mode or risk changes.

## Contextual rigor invariant — 2026-10-01

The correct engineering response is the one that fits the actual goal, time
pressure, failure cost, reuse expectation, lifetime, blast radius and recovery
options. More abstraction, more validation, more ceremony or more CLI flags are
not automatically better.

Every extra mechanism/flag should be explainable as a trade: what behavior it
changes, what risk/value it buys, what it costs, and why that cost is justified
here.

Use the ladder analogy as a durable mental model: a carefully finished reusable
ladder and a minimally sufficient emergency ladder can both be correct in
different contexts. Context chooses the rigor.

## Motivation slogan — 2026-10-01

> Bạn không cần một kế hoạch hoàn hảo.
> Bạn không cần cảm thấy hoàn toàn sẵn sàng.
> Bạn chỉ cần bắt đầu.
> Tiến bộ đến từ hành động, không phải sự hoàn hảo.
> Bạn vẫn đang chờ đợi gì để làm cho hoàn hảo trước khi bắt đầu?

Durable meaning:

```text
DO NOT WAIT FOR PERFECT READINESS
START FROM THE CURRENT REALITY
ACT
OBSERVE
LEARN
IMPROVE
```

This slogan reinforces Matching Start and contextual rigor: planning and quality
matter, but they must not become a reason to avoid the first real action from
which evidence and experience can emerge.

## Response-shape invariant — 2026-10-01

Canonical protocol: `docs/continuity/RESPONSE-PROTOCOL.md`.

Keep separate:

```text
what the user needs to inspect/understand/do
!=
what evidence the assistant needs returned
```

For source comprehension, prefer IDE navigation and reading. Use terminal grep/find/cat only when they materially improve search, system-state inspection or concise evidence collection. Ask for the smallest useful evidence rather than large pasted outputs.

## Comprehension-check preference — 2026-10-01

Do not require the user to explain back every source file/function after reading it.
Assume active IDE reading is happening. Ask for an explanation only when it affects a
real decision, risk, debugging step, or when the user explicitly wants explanation practice.

The user will ask when something is unclear.

## Core-understanding invariant — 2026-10-01

Real problems anchor practice but do not limit the scope of understanding. For foundational or recurring mechanisms, learn the definition, essential properties, state/lifecycle, guarantees, limits, failure modes, neighboring mechanisms and trade-offs so the knowledge transfers beyond one case.

Key distinction:

```text
smallest responsible source change
!=
smallest possible understanding
```

Pressure-driven architecture and foundation-driven learning must stay connected.

## Preferred collaboration shape — 2026-10-01

The user explicitly confirmed that the best interaction pattern is:

```text
real current pressure
 -> core mental model
 -> why the next action is justified
 -> one real user-typed action
 -> explain the state transition
 -> request only minimum evidence
 -> continue from observed reality
```

Do not drift back toward terminal worksheets, forced summaries, full-solution
generation, or shallow case-only explanations. Keep theory transferable but
grounded in real source/runtime work.

## Current persistence/source reality — 2026-10-02

The temporary recovery branch has been promoted to active development branch
`dev`. Before the continuity update, `dev` pointed at
`5fe5cbda5d4be5e54e9b21323efc01d04f9e0e7f` and was 2 commits ahead / 0 behind
`main`.

Live source now proves that the project crossed the previous conceptual boundary:

```text
pgx/v5 v5.11.0 is in go.mod
sqlc.yaml exists and targets pgx/v5
db/query/listing.sql exists
generated db/sqlc code exists
migration 000005 exists
```

Older statements that pgx/sqlc implementation has not happened are stale. What
remains unproven is current runtime wiring/reproducibility: re-run
`sqlc generate`, inspect the diff, establish or locate pgxpool ownership and
execute the first real generated query.

The persistence learning gate completed the core models for database/sql vs
driver, lib/pq vs native pgx, pgx package boundaries, query protocol, prepared
statement caching, transaction connection pinning, pool lifecycle and pool
observability.

The next technical pressure is to reconcile generated persistence artifacts with
live source and produce runtime evidence for the first sqlc + pgxpool query while
containing pgx/sqlc representations inside the Postgres persistence boundary.

## Canonical database finalization milestone — 2026-10-06

The database redesign has moved from the long v0 semantic discovery sequence to a
durable full target package on branch
`architecture/canonical-database-final`, created from
`dev@2a84561cb29dca7f8b198e61d4ad5a6980c2de5e`.

Canonical package:

```text
docs/database/final/README.md
docs/database/final/BDSPro-CANONICAL-DATABASE.dbml
docs/database/final/CANONICAL-DATABASE-ACCEPTANCE.md
docs/database/final/CANONICAL-DATA-DICTIONARY.md
docs/database/final/LEGACY-MIGRATION-MAP.md
docs/database/final/CONTINUATION-PROMPT.md
```

Target scale/proof:

```text
146 canonical target tables
223 parsed FK refs
0 refs to undefined tables
```

Durable semantic anchors:

- Party/Person/Organization are business identity; Account is digital identity.
- Invitation is never the same fact as Membership/Participation.
- authority/capacity is separated from business relationship.
- Organization owner points to Membership; Branch manager/assignment also points
  to Membership.
- Property, Listing and Asset are distinct facts.
- Listing can represent multiple Properties.
- Deal context is Organization or Group; optional Branch is secondary scope.
- Deal Participation points to Person; Customer/Partner may overlap; Admin and
  Lead are separate authority dimensions.
- commission terms != earned commission.
- investment commitment != actual investment.
- money always includes currency.
- generic subject/type IDs are allowed only in non-authoritative operational
  evidence, not canonical business relations.
- planning/GIS preserves PostGIS geometry.

Rejected patterns:

```text
owner_type + owner_id
generic type + id
universal status
universal soft delete / BaseEntity
is_owner shadow flags
single role/status blobs
EAV for stable core facts
generic attachments as business truth
```

The target package does not replace current live migrations/source. Implementation
must be incremental through expand/backfill/compare/prove/switch/contract.

The next database pressure is a full human review of the target. Only after that
review should the first implementation gate be selected from live `dev`.

