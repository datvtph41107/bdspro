# BDSPro

Clean-slate Go backend project.

## Status

This repository is the active implementation authority for the BDSPro rebuild.

The previous repository `datvtph41107/bdspro-backend` is preserved as historical
evidence only. Its architecture assumptions, package layout, service topology and
mentor-derived decisions are not automatically authoritative here.

Current implementation direction:

```text
Go
PostgreSQL / future PostGIS
versioned SQL migrations
explicit SQL
sqlc
pgx/v5 / pgxpool
```

## Current principle

**Canonical repository working contract:** [First-principles main/dev, local ↔ GitHub, PostgreSQL and evidence](docs/continuity/REPOSITORY-WORKING-CONTRACT.md). `main` has been restored from its last historical commit (not yet a verified release); `dev` remains the user's hands-on coding branch. Work is user-authored, then tested against real output, reviewed and intentionally committed.

**Daily tooling reflex:** [Engineering Workflow](docs/continuity/ENGINEERING-WORKFLOW.md). Local uncommitted changes, GitHub commits and PostgreSQL schema are **different states**. No one should auto-overwrite local work.


**Practical engineering standard:** [Proportional Engineering — earn complexity only when it buys real value](docs/continuity/PROPORTIONAL-ENGINEERING-STANDARD.md). Use it to choose the smallest sufficient SQL, migration, Git, Go, test and operations practice for the current risk and product stage. Hands-on SQL/Go/CI remain user-authored per [Practice Protocol](docs/continuity/PRACTICE-PROTOCOL.md).

Build from observed problems and runtime/database behavior:

```text
problem
  -> mechanism
  -> smallest responsible change
  -> real output/proof
  -> explanation
  -> decision
  -> next pressure
```

Do not create architecture folders, packages, frameworks or infrastructure before
the problem that justifies them is understood.

## Repository identity

Current source host:

`github.com/datvtph41107/bdspro`

Current Go module path:

`github.com/datvtph41107/bdspro`

Product identity remains **BDSPro** and is independent from the current personal
GitHub owner namespace.

## Canonical database design

The completed v0 → v1 → v2 → final database target package lives at:

`docs/database/final/README.md`

It includes:

- full DBML source for dbdiagram.io;
- final acceptance/reasoning report;
- complete table data dictionary;
- legacy-to-canonical migration map;
- fresh-chat continuation prompt.

The target DBML is a design/review baseline, **not an all-at-once executable
migration**. Current source/migrations remain implementation truth until each
canonical slice is implemented and proved incrementally.

## Local workspace

Preferred local path:

`~/projects/bdspro`

## Durable continuity

Fresh contexts start from:

`docs/continuity/CONTINUATION-PROMPT.md`

That document owns the recovery order. Live Git/source and reproducible proof
outweigh durable prose whenever they disagree.
