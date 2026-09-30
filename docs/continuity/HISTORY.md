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
