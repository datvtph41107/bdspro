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
