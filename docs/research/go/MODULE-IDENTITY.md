# Go Module Identity

## Core distinctions

```text
Git repository
!= Go module
!= Go package
```

A Git repository is a source/history container.

A Go module is identified by the `module` directive in `go.mod`.

A package is source code compiled together and normally imported through a package
path derived from the module path plus its directory inside the module.

## Common formula

For a module:

```text
module github.com/acme/project
```

and a package directory:

```text
auth/token
```

the package import path is:

```text
github.com/acme/project/auth/token
```

The source may declare:

```go
package token
```

and callers normally use:

```go
token.Parse(...)
```

## Identity is not necessarily physical hosting

Moby:

```text
Git repository: github.com/moby/moby
root module:    github.com/moby/moby/v2
client module:  github.com/moby/moby/client
```

Terraform:

```text
Git repository: github.com/hashicorp/terraform
module:         github.com/hashicorp/terraform
```

Prometheus:

```text
Git repository: github.com/prometheus/prometheus
module:         github.com/prometheus/prometheus
```

Kubernetes:

```text
Git repository: github.com/kubernetes/kubernetes
module:         k8s.io/kubernetes
```

CockroachDB:

```text
Git repository: github.com/cockroachdb/cockroach
module:         github.com/cockroachdb/cockroach
```

Kubernetes proves that a Go module path is a canonical software identity, not
necessarily a literal GitHub URL.

## Ownership rule

Do not invent an organization or vanity domain merely to make a module path look
professional.

A long-lived module identity should be rooted in a namespace that is actually
controlled:

- an owned GitHub/GitLab organization;
- an owned domain with correct Go module discovery;
- or a personal namespace intentionally accepted as the canonical owner.

## Current BDSPro status

Physical Git host:

`github.com/datvtph41107/bdspro`

Canonical module identity:

`OPEN`

No `go.mod` is created yet because the current personal GitHub namespace has not
been accepted as the long-term software owner.
