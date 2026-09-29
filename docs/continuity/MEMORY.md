# Durable Memory

## Project state

BDSPro now continues in a new clean repository:

`datvtph41107/bdspro`

Preferred WSL workspace:

`~/projects/bdspro`

The new repository was empty before bootstrap on 2026-09-29.

Initial repository commit:

`0927d48b444813a8a02dbb5d8ddbb02162c442ed`

## Why a new repository exists

The previous workspace carried:

- a large historical multi-module Go workspace;
- old service/package topology;
- residual untracked directories;
- architecture assumptions created before the current first-principles learning
  process;
- cognitive noise when trying to learn Go from a minimal starting point.

The clean repository removes those accidental constraints.

The old repository remains useful for later comparison, migration evidence and
real production failure examples, but it is not the design authority for the new
repository.

Historical source at transition:

`datvtph41107/bdspro-backend`
`architecture/rebuild-v2@c1245796e320d533ca29d7b92bb23fa28e8380a1`

## Learning objective

The user wants to become able to work naturally as a Go/backend engineer rather
than memorize commands or patterns.

Default method:

- type commands/code personally;
- understand every command before moving on;
- predict important output before running;
- return real output for inspection;
- learn mechanism before abstraction;
- connect language behavior to real engineering consequences;
- use failures as evidence;
- progress from the smallest program toward production concerns.

## Explanation rule

Every meaningful command or code fragment should explain:

```text
WHY NOW?
WHAT DOES THIS EXACTLY DO?
HOW DOES THE MECHANISM WORK?
WHAT STATE CHANGES?
WHAT SHOULD BE OBSERVED?
WHAT CAN FAIL?
WHY DOES THIS MATTER LATER?
```

## Current Go environment observed before repository reset

From the previous WSL environment:

```text
Go toolchain: go1.27.1 linux/amd64
GOROOT: /usr/local/go
GOPATH: /home/bop/go
GOMODCACHE: /home/bop/go/pkg/mod
GOPROXY: https://proxy.golang.org,direct
```

These are machine facts, not automatically project version policy.

## Go concepts already established

The user has already understood the basic distinction between:

- Git repository;
- Go workspace (`go.work`);
- Go module (`go.mod`);
- Go package;
- package/import path;
- package name;
- module cache and proxy.

Important theorem:

```text
Git repository != Go module != Go package

module path = canonical module identity
package import path = module path + package directory within the module
package name = identifier declared by "package <name>" and used by callers
source hosting location may match module identity, but does not have to
```

Examples already examined:

- Moby: `github.com/moby/moby/v2` plus separate
  `github.com/moby/moby/client`;
- Terraform: `github.com/hashicorp/terraform`;
- Prometheus: `github.com/prometheus/prometheus`;
- Kubernetes: source on GitHub but module identity `k8s.io/kubernetes`;
- CockroachDB: `github.com/cockroachdb/cockroach`.

## Canonical module identity is still open

Current repository host:

`github.com/datvtph41107/bdspro`

The GitHub account currently has no organization membership available through the
connected account.

The user does not want a long-lived commercial software identity to be tied to a
personal username merely because the current Git repository is hosted there.

Therefore no `go.mod` should be created until a real canonical namespace is
chosen.

Valid future directions include:

- a GitHub organization actually owned/controlled by the project;
- a domain actually owned/controlled by the project;
- the personal GitHub path if the personal account is intentionally accepted as
  the canonical owner.

Never invent an organization or vanity domain that is not controlled.

## Foundational references

Primary:

- Go specification
- Go modules reference
- Organizing a Go module
- Effective Go
- Go Code Review Comments
- Go Blog: Package names

Production/style evidence:

- Go standard library
- Moby
- Terraform
- Prometheus
- Kubernetes
- CockroachDB
- Uber Go Style Guide
- Google Go Style Guide

Use these problem-first, not as architecture authority.
