# Durable History

## H0 — Previous BDSPro repository

BDSPro previously lived in:

`datvtph41107/bdspro-backend`

It accumulated a large Go multi-module/service topology and extensive architecture
work.

The last Rebuild V2 state before the clean reset was:

`architecture/rebuild-v2@c1245796e320d533ca29d7b92bb23fa28e8380a1`

That repository remains preserved for historical evidence.

## H1 — Learning reset

During first-principles Go study, the existing repository introduced unnecessary
context:

- root `go.work` coordinated many historical modules;
- the greenfield directory was still nested inside that workspace;
- old service/source organization influenced naming and architecture discussions;
- residual untracked service directories existed locally;
- previous mentor-derived architecture assumptions had not all been independently
  re-derived from current problems.

The user chose to remove this bias rather than keep designing around it.

## H2 — Clean repository

On 2026-09-29 the existing empty GitHub repository:

`datvtph41107/bdspro`

was selected as the new active development repository.

The user's preferred local directory is:

`~/projects/bdspro`

The root README was created as the first repository commit:

`0927d48b444813a8a02dbb5d8ddbb02162c442ed`

The new repository begins with no Go architecture and no Go module identity.

That absence is intentional.

## H3 — Retained knowledge

Only durable, independently useful knowledge is carried forward:

- hands-on terminal/code learning protocol;
- Git/source/proof continuity discipline;
- Go language/module/package concepts already verified;
- official Go references;
- production OSS observations;
- package/module naming research.

Old architectural conclusions must be re-proven before adoption.


## H4 — Matching Start durable recovery

On 2026-09-29 the working method itself became a durable repository artifact.

The user identified the successful collaboration pattern as **Matching Start —
BDSPro Refactor Mindset**: begin from the smallest understood current state,
surface the next real requirement/pressure, and introduce only the smallest
boundary that has earned its cost.

This was separated from the moving checkpoint so future architecture evolution
cannot silently erase the reasoning method.

Added:

- `docs/continuity/MINDSET.md` — stable engineering/training method;
- `docs/continuity/CONTINUATION-PROMPT.md` — single fresh-context bootstrap.

The recovery contract requires live Git/source/proof reconciliation, complete
source-tree inventory, mindset recovery and checkpoint recovery before new
architecture/source decisions.


## H5 — GitHub owner namespace collision

On 2026-09-30 PostgreSQL became the first requirement that needs a third-party Go
dependency, making canonical module identity an operational rather than purely
semantic concern.

The user attempted to create the preferred GitHub organization `bdspro`, but
GitHub reported that the name was already taken.

This invalidated the assumption that durable product identity and source-host
owner namespace must be identical.

The durable refinement is:

```text
product/system identity = BDSPro
repository name         = bdspro
source-host owner       = independently chosen controlled namespace
Go module path          = derived after that namespace is proven
```

BDS-013 was superseded rather than erased. BDS-017 preserves the product/repo
identity, while BDS-018 tracks the still-open hosting namespace.


## H6 — Organization gate intentionally deferred

After GitHub reported that the exact organization name `bdspro` was already
taken, the user chose not to make organization creation a prerequisite for the
current learning/build phase.

Matching Start was applied directly: organization ownership currently solves no
required runtime or dependency problem, while the existing repository
`datvtph41107/bdspro` is real and controlled.

For the present phase the operational module path is therefore:

`github.com/datvtph41107/bdspro`

The BDSPro product identity remains independent of that hosting namespace.
Organization or vanity-domain migration is deferred until a concrete
ownership/publishing/collaboration pressure appears.


## H7 — PostgreSQL identity/bootstrap reset

The initial Docker/PostgreSQL experiment used generic `postgres/postgres`
credentials to expose connectivity and lifecycle pressure. The user explicitly
rejected carrying that shortcut forward because it hides important identity and
privilege semantics.

The local datastore bootstrap was therefore reset conceptually before any real
application data existed.

The durable model now separates:

```text
container runtime identity
persistent-volume identity
logical database identity
cluster/bootstrap administrator identity
application login identity
```

The application must not run as the PostgreSQL bootstrap superuser. A future
migration identity is intentionally not created until schema-change pressure
earns it.


## H8 — Runtime, data, concurrency and evidence checkpoint

By 2026-10-01 the clean BDSPro learning path had advanced well beyond the initial
PostgreSQL bootstrap checkpoint.

The work was intentionally derived pressure-by-pressure rather than from a target
architecture:

```text
minimal HTTP process
 -> explicit lifecycle
 -> required PostgreSQL startup dependency
 -> Dockerized local datastore
 -> liveness/readiness split
 -> first durable Listing state
 -> schema migrations + dirty-state reasoning
 -> DRAFT/PUBLISHED transition
 -> concurrency-safe conditional mutation
 -> business transaction across multiple durable facts
 -> rollback proof
 -> Listing package boundary
 -> publish completeness and non-editability
 -> concurrent Edit/Publish proof
 -> Publication Readiness and TOCTOU
 -> Bash/Docker/testing foundations
 -> real-PostgreSQL integration-evidence pressure
```

A dedicated acceptance artifact was added:

`docs/continuity/LEARNING-CHECKPOINT-2026-10-01.md`

The moving `CHECKPOINT.md` and `MEMORY.md` were refreshed so future recovery
does not return to the obsolete "PostgreSQL is just arriving" state.

The next authorized pressure is safe, repeatable orchestration of the
integration-test environment. The smallest candidate response is one
project-local Bash workflow; Makefile, CI, Testcontainers and Repository/Store
abstractions remain unearned until stronger pressure appears.

## H9 — Active coding and source-trace reset

On 2026-10-01 the user identified a major training failure mode: receiving full
functions/tests/files before reasoning about them encourages copy/paste fluency
rather than engineering fluency.

The working contract was therefore strengthened. BDSPro learning now explicitly
requires the user to derive responsibilities, read relevant primary docs/source,
write implementation personally, run it, debug real output and explain the
mechanism in their own words.

Added:

- `docs/continuity/PRACTICE-PROTOCOL.md`.

The continuation and method documents were updated so future contexts recover
this rule before implementation.

This milestone also corrected an over-specific checkpoint assumption: the next
integration-workflow response is no longer precommitted to one Bash script.
Recurring project capabilities may earn Make targets; Bash is introduced only
where imperative/safety logic earns it. The user will implement those boundaries
incrementally rather than paste a completed workflow.

## H10 — BDSPro becomes a developer-immersion practice system

On 2026-10-01 the user broadened the active-coding reset into a long-horizon
professional training objective. BDSPro will deliberately repeat foundational
shell, Git, Docker, PostgreSQL, Go, testing and source-tracing work under
increasingly realistic failure and operational pressure.

The aim is transferable engineering fluency: recognize problems, locate primary
sources, type and operate primitives naturally, diagnose failures, reason from
multiple role perspectives and later describe only work actually performed in CV
and interview contexts.

Added `docs/continuity/DEVELOPER-IMMERSION.md` and made it part of recovery.

## H11 — Daily engineering and command reflex standardized

On 2026-10-01 the user required a concrete daily operating standard so deliberate practice
would become a working reflex rather than a collection of lessons. Added
`docs/continuity/ENGINEERING-WORKFLOW.md` covering first-encounter command tracing,
state/risk classification, primary-source reading, stack-specific Git/Bash/Docker/PostgreSQL/Go
reflexes, branch/commit reasoning and the AI response contract.

The model intentionally becomes faster with repetition: familiar primitives are abbreviated,
while new flags, environments, failures or destructive operations reopen the full trace.

## H12 — Contextual rigor and flag discipline

On 2026-10-01 the user refined the mindset with a ladder analogy: one engineer may
build a polished reusable ladder, another may build the minimum sufficiently safe
ladder to escape quickly. The right answer depends on time pressure, failure cost,
reuse, lifetime and surrounding threats.

This was generalized into a durable rule: engineering rigor must be proportional
to actual context, risk and value. Extra abstractions, safety checks, workflow
steps and CLI flags must earn their cost rather than being retained from examples
or convention.

The rule was added to MINDSET.md, DEVELOPER-IMMERSION.md,
ENGINEERING-WORKFLOW.md and PRACTICE-PROTOCOL.md.

## H13 — Recovery implementation promoted to dev and SQL persistence foundation materialized

On 2026-10-02 the preserved recovery implementation was committed and pushed as:

`5fe5cbda5d4be5e54e9b21323efc01d04f9e0e7f — ft: sqlc sql base`

A new `dev` branch was created at that exact commit. `main` remained the
stable/default baseline at
`1a49d5bc469110c3b4b8d42a0d5f419bc45857bc`, so the transition preserved history
without reset, rebase or force-push.

This materialized source changed the persistence checkpoint substantially:

- Makefile project operations now exist;
- migrations exist through 000005, including nullable non-negative Listing price;
- sqlc configuration and query source now exist;
- generated sqlc persistence code now exists;
- go.mod now contains pgx/v5;
- generated code targets native pgx/v5 and exposes pgx/pgconn/pgtype persistence
  representations;
- the committed snapshot removed the former Compose/integration-script files.

The previous durable statement that pgx/sqlc implementation had not yet been
verified is superseded by live Git/source.

The next proof is runtime/reproducibility rather than another conceptual library
selection: verify `sqlc generate` is deterministic, locate or establish the
shared pgxpool ownership point and execute the first generated query while keeping
driver/generated representations inside the intended persistence boundary.
