# Current Checkpoint

Updated: 2026-09-30

## Source identity

Repository currently hosted at:

`datvtph41107/bdspro`

Branch:

`main`

Always resolve the live HEAD and preserve uncommitted local work before mutation.

## Current phase

```text
B1 — Matching Start / Source Growth From First Principles
STATUS: ACTIVE
```

Governing method:

`docs/continuity/MINDSET.md`

## Proven runtime learning

The current hands-on sequence has established:

- a minimal root `main.go` can own a small HTTP process without premature
  source hierarchy;
- configurable `PORT` adds configuration responsibility without by itself
  justifying a package;
- graceful shutdown exposed real process/runtime lifecycle responsibility;
- an in-flight slow request completed after SIGINT before process exit, proving
  graceful shutdown behavior;
- separating application HTTP construction behind a small function is a weaker
  boundary than immediately creating directories/packages;
- explicit listener acquisition makes startup truth depend on successful OS
  resource acquisition rather than intent/log ordering;
- resource construction, acquisition, use and release are distinct lifecycle
  concepts.

Local source may be ahead of tracked `main`; recover it from the actual
workspace before changing source.

## New requirement

```text
BDSPro needs PostgreSQL as a durable datastore.

The process must not become operationally ready until the required datastore is
usable, and the database resource must have an explicit process lifetime.
```

## Pressure exposed before implementation

PostgreSQL introduces the first third-party Go dependency.

```text
PostgreSQL requirement
    -> pgx/pgxpool
    -> external module dependency
    -> go.mod becomes operationally necessary
    -> canonical module identity must be real
```

Therefore the previously deferred module-identity decision is now an active gate.

## New evidence — GitHub namespace collision

The user attempted to create GitHub organization:

`bdspro`

GitHub reported:

```text
The name 'bdspro' is already taken.
```

This disproves the earlier operational assumption that the durable product name
must also be the exact GitHub owner namespace.

## Refined identity model

```text
PRODUCT / SYSTEM IDENTITY
BDSPro

REPOSITORY NAME
bdspro

SOURCE-HOST OWNER
OPEN — must be controlled and durable

GO MODULE PATH
OPEN — derived only after a real controlled namespace is chosen
```

Product identity and hosting identity are now explicitly separate concerns.

Do not rename the product or repository merely to work around a GitHub namespace
collision.

## Current open decisions

### BDS-018 — durable source-host owner namespace

The replacement owner should:

- be controlled by the project/user;
- be durable enough for long-lived source hosting;
- remain neutral to language, framework, runtime topology and deployment shape;
- not encode temporary labels such as backend, service, microservice, v2, Go,
  dev or staging merely to obtain availability.

### BDS-004 — canonical Go module path

Remains OPEN until BDS-018 is closed or a controlled vanity-domain import path is
deliberately chosen.

## Current question

```text
What durable namespace should own the source identity now that the exact GitHub
owner "bdspro" is unavailable?
```

This is not a branding exercise. It is a source-identity and dependency-management
gate created by the first real third-party dependency.

## Next authorized work

1. choose/prove a durable controlled owner namespace or controlled vanity domain;
2. update repository hosting only if that choice requires it;
3. inspect any existing local untracked `go.mod` rather than overwriting it;
4. initialize/commit canonical `go.mod`;
5. only then add pgx and continue PostgreSQL resource-lifecycle pressure.

## Still not earned

Do not create merely because PostgreSQL is arriving:

- `database/`;
- `repository/`;
- `storage/`;
- ORM abstractions;
- service layers;
- `internal/`;
- `cmd/`;
- readiness/liveness policy beyond the current startup requirement.

Those require their own demonstrated pressure.
