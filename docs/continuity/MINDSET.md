# Matching Start — BDSPro Refactor Mindset

## Purpose

This document preserves the long-lived engineering and training mindset for
BDSPro. It is intentionally more stable than any particular architecture,
folder tree, framework, service topology or checkpoint.

The objective is not to memorize a target architecture. The objective is to
become able to derive architecture again from first principles when the system,
requirements and constraints change.

## North star

```text
Do not begin from the architecture we want.

Begin from the smallest current state that is understood and working.
Make the next real pressure visible.
Introduce only the smallest boundary that solves that pressure.
```

Compressed:

```text
START SIMPLE
MAKE PRESSURE VISIBLE
EVOLVE DELIBERATELY
```

## Matching Start reasoning chain

Every substantial design discussion should be recoverable through this chain:

```text
CURRENT REALITY
    ↓
INTENT — what are we trying to achieve?
    ↓
NEW REQUIREMENT — what changed?
    ↓
PRESSURE — where does the current form start to hurt?
    ↓
INSUFFICIENCY — why is the current form no longer enough?
    ↓
SMALLEST RESPONSIBLE RESPONSE
    ↓
EARNED BOUNDARY
    ↓
MECHANISM — how does Go/Git/runtime/database/network represent it?
    ↓
EVIDENCE — official docs, OSS, old BDSPro, experiment
    ↓
TRADE-OFF — what value and cost are accepted?
    ↓
DECISION / SOURCE CHANGE
```

A proposed directory, package, module, process, service, database, repository or
framework that cannot be traced through this chain is not yet justified.

## First-principles questions

For an important mechanism or boundary, ask:

```text
WHAT is it?
WHY does it exist?
WHAT problem does it solve?
WHAT happens without it?
WHEN does it become necessary?
WHAT boundary does it create?
WHAT does that boundary buy?
WHAT does that boundary cost?
WHY this name?
WHY this location?
WHY now?
```

"People usually do this", "best practice", "Clean Architecture says so", "DDD
says so", or "this OSS repo has it" is evidence at most, never the root cause.

## Do not skip levels

Prefer the weakest boundary that solves the demonstrated problem.

```text
file
  -> package
  -> stronger visibility/API boundary
  -> Go module
  -> process
  -> independently deployed/networked service
  -> repository
```

This is not a mandatory extraction sequence. It is a reminder that stronger
boundaries impose stronger costs. A higher boundary must buy value that the
current one cannot provide.

Do not split merely in anticipation of hypothetical scale. Also do not preserve
a weak boundary dogmatically after real pressure demonstrates the need for a
stronger one.

## Source-tree principle

```text
SOURCE TREE IS A MAP OF THE SYSTEM,
NOT A CATALOG OF ARCHITECTURE PATTERNS.
```

A root item or package should earn its existence through a real repository,
runtime, interface, visibility, subsystem or business responsibility.

Do not pre-create `cmd/`, `internal/`, `pkg/`, `services/`, `domain/`,
`repositories/`, `utils/`, or any other conventional shape merely because a
template or mature repository contains it.

## Training principle

Learning and implementation are one process.

The user should build working reflexes rather than memorize commands or final
forms. For meaningful work:

```text
UNDERSTAND
  -> PREDICT
  -> TYPE / IMPLEMENT
  -> OBSERVE REAL OUTPUT
  -> EXPLAIN MECHANISM
  -> CONNECT TO ENGINEERING CONSEQUENCE
  -> GENERALIZE
```

Explanations should connect immediate action to mechanism and architectural
consequence, without turning every step into a microscopic command/commit cycle.

Work in meaningful knowledge/architecture blocks. Zoom into individual
commands/files only when they carry a mechanism worth understanding or when the
user asks.

## Evidence roles

### New BDSPro source

The active repository is the materialized result of earned decisions. Do not use
it as a playground for speculative architecture.

### Old BDSPro source

Historical source is a case-study database:

- real requirements;
- production-ish flows;
- previous mistakes;
- useful mechanisms;
- migration and scaling pressures.

It is not current architecture authority.

### Mature OSS

Moby, Prometheus, etcd, Terraform, NATS, Temporal, Caddy, Traefik, Kubernetes,
CockroachDB and similar repositories are pressure-tested evidence.

For a structure found there, ask:

```text
What problem exists there?
What pressure caused this boundary?
What mechanism solves it?
Does BDSPro have the same pressure now?
What would copying it cost?
What is the smallest experiment that would justify it here?
```

### Official sources

For language/tool semantics, prefer the Go specification and official Go
documentation before repository convention.

## Anti-drift checks

Stop and return to the current checkpoint if any of these happen:

- a final folder tree is proposed before a current pressure is established;
- business nouns are automatically converted into services;
- service/process/repository/module are treated as the same boundary;
- a mature OSS tree is copied because it looks professional;
- legacy BDSPro topology is treated as inherited truth;
- a "best practice" is used as the cause rather than evidence;
- many empty architecture directories appear before code pressure;
- discussion jumps multiple abstraction levels without explaining why;
- implementation details replace the actual intent/problem;
- work degrades into tiny file/command/commit cycles that lose the larger
  learning target.

## Current canonical example of the mindset

The present training question is intentionally simple:

```text
A Go program begins with root main.go.

How does it grow without turning main.go or the repository root
into an implementation dump?
```

The correct approach is not to answer immediately with `cmd/`, `internal/` or
a final architecture.

Instead observe the growth:

```text
small working main.go
    -> new requirement
    -> main.go accumulates responsibility
    -> split files if that is enough
    -> responsibility/dependency pressure becomes visible
    -> only then consider a package/directory boundary
    -> only later consider stronger boundaries if new pressure appears
```

This example is the reference shape for future reasoning even when the topic
changes from source organization to database, APIs, runtime, concurrency,
messaging, deployment or scaling.

## Durable rule

The mindset may evolve when stronger evidence appears, but it must never change
silently.

If it changes:

1. state what pressure invalidated the previous rule;
2. record evidence;
3. update this document;
4. record the decision/history;
5. keep the superseded reasoning discoverable.
