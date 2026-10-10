# Matching Start — BDSPro Refactor Mindset

> **2026-10-10 DURABLE CLARIFICATION — SHOW THE CAUSE, NOT JUST THE CLAIM.**
> Before significant design decisions, read [EVIDENCE-DRIVEN-REASONING.md](EVIDENCE-DRIVEN-REASONING.md). Each comparative conclusion must be visualizable through a real/hypothetical actor, a specific operation, tiny rows/data and a state transition; backed by actual source, primary documentation or reproducible output; compared fairly with alternatives and counterexamples; qualified with what remains unproven. "Simpler/safer/more scalable" without observable mechanics is an empty judgment. Do not confuse invented use cases with market proof or canonical candidate schema with validated business need. This **extends** Matching Start: the user learns by executing and falsifying actual BDSPro changes, not accepting ready-made conclusions.

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

## Contextual rigor — fit the solution to the situation

Engineering quality is contextual. The most polished, reusable or defensive
solution is not automatically the correct one.

For every meaningful decision, evaluate the actual situation:

```text
GOAL
  -> TIME PRESSURE
  -> FAILURE COST
  -> SAFETY REQUIREMENT
  -> EXPECTED LIFETIME
  -> REUSE EXPECTATION
  -> NUMBER OF USERS / ACTORS
  -> CHANGE FREQUENCY
  -> OBSERVABILITY / RECOVERY OPTIONS
  -> COMPLEXITY COST
```

Then choose the smallest level of rigor that satisfies the real need.

A useful analogy is building a ladder to escape a room. One approach may build a
carefully finished ladder optimized for repeated safe use; another may build the
minimum sufficiently strong ladder to escape quickly. Neither is inherently
better. The correct choice depends on whether the ladder will be reused, how much
time exists, how costly failure is, whether there is an immediate threat, and
what happens after escape.

The same applies to software:

```text
more abstraction != more correct
more flags       != more professional
more validation  != always more valuable
faster hack      != always irresponsible
```

The decision must be tied to context, risk and value.

For a command, option, abstraction or safety mechanism, ask:

```text
What concrete problem does this extra thing solve here?
What changes if I omit it?
What new failure does it prevent?
What complexity or cognitive cost does it add?
Is that trade worth paying in this situation?
```

Do not keep syntax, flags or layers merely because they appeared in an earlier
example. Re-evaluate them against the current intent.

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

## Core model before local exploitation

A real problem is an anchor for learning, not the maximum scope of learning.

When a mechanism is important, do not learn only the fragment needed to pass
the current case. Recover its core model:

```text
DEFINITION
  -> ESSENTIAL PROPERTIES
  -> STATE / LIFECYCLE
  -> GUARANTEES
  -> NON-GUARANTEES / LIMITS
  -> FAILURE MODES
  -> RELATION TO NEIGHBORING MECHANISMS
  -> COST / TRADE-OFF
  -> HOW TO OBSERVE IT
  -> HOW THE CURRENT CASE USES IT
```

The current requirement determines where to enter and how deep to go first,
but it must not reduce an important mechanism to a one-off recipe.

Pressure-driven architecture and foundation-driven learning are complementary:

```text
PRESSURE decides WHAT must change in the system.
CORE MODEL decides WHAT must be understood to work with that mechanism well.
```

Do not confuse 'smallest responsible source change' with 'smallest possible
understanding'.
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

## Active skill formation

Matching Start applies to the learner as well as the architecture.

Do not begin from the finished code we want. Begin from the smallest code the
user can reason about and write personally, expose the next gap in understanding,
and add only the smallest hint needed to move forward.

The default coding progression is:

```text
UNDERSTAND REQUIREMENT
  -> RESEARCH UNKNOWN MECHANISM
  -> PREDICT
  -> WRITE BY HAND
  -> RUN
  -> READ FAILURE/SUCCESS
  -> DEBUG
  -> EXPLAIN IN OWN WORDS
  -> REPEAT
  -> ONLY THEN ABSTRACT
```

A green program produced by copying a complete supplied implementation is not
sufficient learning evidence. The user must build recall, naming, decomposition,
documentation-reading and debugging reflexes.

The detailed operational contract is
`docs/continuity/PRACTICE-PROTOCOL.md` and is mandatory unless the user
explicitly switches modes for a particular task.

## Professional immersion

The repository is also a deliberate-practice environment. Fundamental tool and
runtime skills must recur until they become working reflexes rather than isolated
lessons.

Architecture training therefore includes repeated real interaction with Git,
Bash/processes, Docker, PostgreSQL/SQL/CLI, Go tooling, tests and failure recovery.
Difficulty should increase only when it exposes a real pressure or risk.

Use multiple reasoning lenses where useful: business/BA, fresher implementation,
middle-level responsibility/invariant design, and senior system/operational
ownership. Complexity is accepted only in exchange for demonstrated value or
risk reduction.

Long-horizon training details are owned by
`docs/continuity/DEVELOPER-IMMERSION.md`.

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
- rigor, abstraction, validation or CLI flags are retained without explaining the
  concrete risk/value they buy in the current context;
- many empty architecture directories appear before code pressure;
- discussion jumps multiple abstraction levels without explaining why;
- implementation details replace the actual intent/problem;
- a complete solution is supplied before the user has had a real opportunity to
  derive and type it;
- official/library documentation is skipped for an unfamiliar mechanism that is
  material to the current pressure;
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
