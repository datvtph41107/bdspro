# Current Checkpoint

Updated: 2026-09-29

## Source identity

Repository:

`datvtph41107/bdspro`

Branch:

`main`

Always resolve the current live HEAD rather than assuming this file contains the
latest commit.

## Current tracked root observed on live main before this checkpoint update

```text
.gitattributes
README.md
docs/
main.go
```

Local-only/uncommitted files are not authoritative until re-inspected in the
actual workspace. Preserve them if present.

## Current phase

```text
B1 — Matching Start / Source Growth From First Principles
STATUS: ACTIVE
```

## Governing mindset

Read:

`docs/continuity/MINDSET.md`

The working rule is:

```text
Do not design the final tree.
Start from the smallest understood working state.
Expose the next real pressure.
Introduce the smallest boundary that solves it.
```

## Closed foundation

- the new `datvtph41107/bdspro` repository is active authority;
- old BDSPro source is historical/case-study evidence only;
- mature OSS and official Go material are evidence, never templates;
- root `main.go` is a valid minimal starting shape;
- repository-root items and source boundaries must earn their existence;
- do not pre-create architecture directories before a real pressure exists;
- source organization should be derived from problem/intent/responsibility, not
  from a desired architecture pattern;
- work should proceed in meaningful reasoning blocks rather than microscopic
  file/command/commit loops.

## Current question

```text
A Go program starts with root main.go.

How does it grow without turning main.go or the repository root
into an implementation dump?
```

This question is being used as the first concrete training case for observing how
architecture emerges from growth pressure.

## Current pressure to study

`main.go` begins accumulating responsibilities.

The next reasoning block must determine:

```text
What is the essential responsibility of an entrypoint?
What still belongs in main.go?
What is application implementation rather than entrypoint work?
What is the first demonstrated pain?
Can a file split solve it?
When does a package/directory boundary become earned?
```

Do not answer this by jumping directly to `cmd/`, `internal/`, a module map,
microservices or a final source tree.

## Matching Start sequence for this checkpoint

```text
minimal main.go
    -> add a real requirement
    -> observe what responsibility accumulates
    -> identify the concrete pressure
    -> try the smallest response
    -> evaluate what it buys/costs
    -> compare official Go / mature OSS only after the pressure is known
    -> record an earned decision if evidence is sufficient
```

## Operational open decision

BDS-004 — exact canonical Go module path remains OPEN until the intended
`bdspro` owner namespace is actually controlled.

The semantic target `bdspro/bdspro` and one-root-module model are already
closed decisions. Do not invent ownership. Reconcile any local untracked
`go.mod` before committing it.

This operational namespace question does not block conceptual/source-organization
learning, but it remains a gate before committing a canonical module identity.

## Next authorized work

Continue the current pressure:

**entrypoint vs application implementation**.

Use a minimal concrete program/requirement to expose the problem, then reason
forward from observed pressure.

## Not authorized by default

Do not pre-design or pre-create:

- `cmd/`;
- `internal/`;
- `pkg/`;
- architecture layers;
- business module/service topology;
- separate processes/services/repos;
- framework/database/messaging infrastructure without a current requirement.

Any of these may become valid later, but only after its pressure is demonstrated.
