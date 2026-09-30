# Current Checkpoint

Updated: 2026-09-30

## Source identity

Active repository:

`datvtph41107/bdspro`

Branch:

`main`

Current operational Go module path for this phase:

`github.com/datvtph41107/bdspro`

This is a source/dependency identity decision for the present phase, not a
renaming of the BDSPro product. Organization/vanity ownership is intentionally
deferred until a real ownership/publishing/collaboration pressure appears.

## Current phase

```text
B1 — Matching Start / Source Growth From First Principles
STATUS: ACTIVE
```

Governing method:

`docs/continuity/MINDSET.md`

## Proven runtime learning

- minimal root `main.go` is sufficient until a real source pressure appears;
- configurable `PORT` adds configuration without automatically earning a package;
- graceful shutdown exposed process lifecycle as a real responsibility;
- an in-flight slow request completed after SIGINT before process exit;
- a function boundary can separate HTTP application construction from process
  lifecycle without prematurely creating packages;
- resource acquisition should be proven before operational success is announced;
- construction, acquisition, use, ownership and release are distinct lifecycle
  concepts.

Local source may be ahead of tracked `main`; recover it from the actual
workspace before changing source.

## Current requirement

```text
BDSPro needs PostgreSQL as its first durable datastore.

The process must fail startup when the required datastore is not usable.
The database pool must have an explicit process lifetime.
```

## Pressure now authorized

PostgreSQL is the first third-party Go dependency.

The organization-name detour is closed for the current phase. The smallest
response is to use the repository path that actually exists:

`github.com/datvtph41107/bdspro`

Before adding pgx:

1. inspect any local untracked `go.mod`;
2. align it to the current operational module path rather than overwriting unknown
   local work;
3. verify the module state;
4. add pgx/pgxpool;
5. keep database construction/verification in `main.go` initially;
6. observe whether dependency/lifecycle pressure actually earns a stronger source
   boundary.

## Current PostgreSQL pressure questions

```text
What does creating a pool actually prove?
What does Ping prove?
When is startup allowed to progress?
Who owns the pool lifetime?
Who receives the pool?
What shuts down first: HTTP or DB?
What failure should abort startup?
What pressure, if any, finally makes main.go know too much?
```

## Not earned yet

Do not create merely because PostgreSQL is arriving:

- `database/`;
- `repository/`;
- `storage/`;
- ORM;
- service layers;
- `internal/`;
- `cmd/`;
- a separate organization;
- readiness/liveness policy beyond the current startup requirement.

Those need their own demonstrated pressure.
