# Current Checkpoint

Updated: 2026-09-29

## Source identity

Repository:

`datvtph41107/bdspro`

Branch:

`main`

Initial bootstrap commit:

`0927d48b444813a8a02dbb5d8ddbb02162c442ed`

Always resolve the current live HEAD rather than assuming this file contains the
latest commit.

## Local state supplied by user

```text
working directory: ~/projects/bdspro
directory contents: empty before clone/bootstrap
```

The remote repository also had size 0 before the bootstrap commit.

## Current phase

```text
B0 — Clean Repository Bootstrap
STATUS: ACTIVE
```

## Closed facts

- a new clean repository is the working authority;
- the previous repository remains historical evidence only;
- root README is the first committed file;
- durable continuity belongs in the new repository;
- external Go/OSS research may be retained as evidence;
- old architecture assumptions are not inherited automatically.

## Current open question

`B0.1 — Canonical Go module identity`

The repository is physically hosted at:

`github.com/datvtph41107/bdspro`

but that personal namespace has not been accepted as the long-term software
identity.

No `go.mod` should be created before this question is resolved.

## Why this gate matters

The module path becomes the prefix of package import paths and can spread through:

- source imports;
- generated code;
- documentation/examples;
- other repositories;
- releases and downstream consumers.

Renaming is cheap now and progressively more expensive after code and consumers
exist.

## Next authorized actions

1. align the empty local directory with the new remote repository;
2. inspect the resulting Git state;
3. decide canonical ownership/namespace from actual controlled identities;
4. create `go.mod` only after that decision;
5. then create the smallest executable Go program.

## Not authorized yet

Do not pre-create:

- `cmd/`;
- `internal/`;
- `pkg/`;
- `domain/`;
- `service/`;
- `repository/`;
- framework/config/database infrastructure.

Those boundaries must emerge from actual code/problem pressure.


## Root-baseline evidence review — 2026-09-29

Evidence reviewed:

- official Go module layout guidance;
- Moby;
- Terraform;
- Prometheus;
- Kubernetes;
- CockroachDB.

Research:

`docs/research/go/ROOT-BASELINE.md`

Closed conclusions:

- root `main.go` is the correct current executable shape;
- do not create `cmd/`, `internal/`, `pkg/` or architecture-layer directories
  before an actual code/visibility/command pressure exists;
- Makefile, Dockerfile and CI are deferred until they own real repeatable
  workflows/contracts;
- `.editorconfig` is removed from the initial baseline;
- minimal `.gitattributes` LF normalization is provisionally justified by the
  real Windows/WSL development environment.

Current clean-root target:

```text
README.md
main.go
go.mod          # still blocked by BDS-004 canonical identity
.gitignore      # minimal/evidence-driven
.gitattributes  # next repository-hygiene experiment
docs/
```

The only blocker before normal module-based development remains BDS-004:
canonical Go module identity.
