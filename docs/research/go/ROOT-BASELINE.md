# Root Baseline — Evidence from Mature Go Repositories

Updated: 2026-09-29

## Question

What should exist at the repository root at the beginning of a clean Go project,
and which files/directories should appear only after real pressure exists?

## Official Go baseline

The official Go module layout guidance explicitly permits a basic executable to
start with only a module file and Go source files in the repository root. Larger
commands may later split supporting code into packages, with `internal/`
recommended when that code is not intended as a public API. `cmd/` is a common
convention for multiple commands or mixed command/library repositories, not a
language requirement.

Therefore a root-level `main.go` is a valid starting structure, not a temporary
hack.

## Production repository observations

### Moby

Root has mature operational/build surfaces including `cmd/`, `internal/`,
`pkg/`, Makefile, Dockerfiles, `.gitattributes`, `.gitignore`, `go.mod` and
`go.sum`.

Important counterexample: Moby's repository `.gitignore` treats
`.editorconfig` as local repository customization rather than shared root
policy.

Lesson:

- mature build/deployment files appear because Moby has mature build/release
  problems;
- `.editorconfig` is not universally required even in a very large Go project;
- multiple module/public API boundaries exist because release/support contracts
  justify them.

### Terraform

Terraform keeps executable Go source files directly at repository root while most
implementation code lives under `internal/`. It also has a Makefile,
Dockerfiles and an explicit Go-version file.

Lesson:

- a large mature project does not prove that every executable must live under
  `cmd/`;
- root `package main` source can remain valid when it fits the project;
- supporting implementation can move behind `internal/` without moving the
  executable merely for aesthetics.

### Prometheus

Prometheus has multiple commands under `cmd/`, large capability packages such
as `scrape`, `rules`, `storage`, and mature Makefile/Docker/build tooling.

Its root `.gitattributes` is currently highly purpose-specific: it marks a
generated golden file for GitHub linguist rather than imposing a universal
editor policy.

Lesson:

- `cmd/` becomes valuable when command surface grows;
- root policy files should exist for a concrete repository need;
- large repositories can organize reusable/capability code differently from a
  self-contained server.

### Kubernetes

Kubernetes has `cmd/`, `pkg/`, `staging/`, `vendor/`, `go.work`, Makefile,
version/tooling metadata and many project-governance files.

Its root `.gitattributes` explicitly enforces LF with:

```text
* text=auto eol=lf
```

and adds merge/generated-file rules.

Lesson:

- line-ending normalization is a concrete cross-platform repository policy;
- large-scale workspace/vendor/staging structure is an evolved response to
  Kubernetes' scale and publication model, not a bootstrap template;
- even Kubernetes states that the root `k8s.io/kubernetes` module is not a
  supported library API for external consumers.

### CockroachDB

CockroachDB has a very large build/development surface: Bazel files, a dedicated
`dev` tool, `pkg/`, build/scripts/tools directories, `.editorconfig`,
`.gitattributes`, GNUmakefile, `go.mod` and `go.sum`.

Its developer documentation explains that `./dev` exists as a wrapper around
Bazel for common workflows.

Lesson:

- custom build tooling should be justified by build complexity;
- a shared `.editorconfig` can be valuable in one repository without becoming a
  universal Go requirement;
- developer command surfaces evolve when raw commands become costly/repetitive.

## Cross-repository conclusions

### Strong claims

1. `go.mod` is the Go module identity/dependency root once the repository becomes
   a module.
2. A simple command may legitimately begin with root `main.go`.
3. `cmd/` is not mandatory. It earns its place when there are multiple commands
   or command/library separation benefits.
4. `internal/` should not be created empty; it becomes valuable when supporting
   packages need a private import boundary.
5. `pkg/` is not a universal Go directory and should not be created merely as a
   place to put packages.
6. `go.sum` is tool-maintained dependency integrity state; do not create or edit
   it as decorative bootstrap content.
7. Makefile/task runners are useful after a repeatable multi-step developer
   workflow exists.
8. Dockerfiles belong to a container build/runtime problem, not to "being a
   production project".
9. CI should prove an existing contract; it should not precede the contract it is
   supposed to prove.
10. Root policy files must solve repository-wide problems, not imitate mature
    repositories.

### Repository hygiene conclusions

`.gitignore`:
- valuable early when local/build/secret artifacts can appear;
- keep entries tied to actual artifact classes;
- editor-specific ignores can be global rather than project policy.

`.gitattributes`:
- not required by Go;
- justified for repository-wide Git behavior such as LF normalization, generated
  file classification, merge drivers or LFS;
- BDSPro has a real cross-platform Windows/WSL pressure, so LF normalization has
  concrete value.

`.editorconfig`:
- useful when the team needs shared editor behavior across languages/editors;
- not required by Go and not universally used by major Go projects;
- Go formatting authority remains `gofmt`;
- defer until a real non-Go/editor consistency problem appears.

## BDSPro clean-root baseline

Current intended root:

```text
bdspro/
├── README.md
├── main.go
├── go.mod            # after canonical module identity is resolved
├── .gitignore        # minimal, evidence-driven entries
├── .gitattributes    # LF normalization justified by Windows/WSL workflow
└── docs/
```

Not part of the current baseline:

```text
.editorconfig
Makefile
Dockerfile
cmd/
internal/
pkg/
config/
domain/
service/
repository/
migrations/
.github/workflows/
```

Those are not forbidden. Each requires the problem that earns it.

## Engineering rule

Do not ask:

```text
"What files do professional Go projects have?"
```

Ask:

```text
"What repository-wide invariant or repeated cost does this file/directory own?"
```

A root item is justified when its absence creates a concrete, recurring or
high-risk problem whose scope is the repository.
