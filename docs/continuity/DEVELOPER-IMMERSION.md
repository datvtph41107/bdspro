# Developer Immersion Model

## Purpose

BDSPro is not only a software project. It is the primary deliberate-practice
environment for building transferable backend-engineering fluency.

The target is not "knowing commands" or "finishing features with AI". The target
is to become a developer who can repeatedly move from an unfamiliar real problem
to a correct working result through first-principles reasoning, primary-source
research, coding, debugging and operation.

Time spent repeating fundamentals is intentional when it produces faster recall,
better diagnosis and stronger judgment later.

## North star

```text
SEE THE PROBLEM
  -> MODEL THE SYSTEM
  -> FIND THE UNKNOWN
  -> TRACE PRIMARY SOURCES
  -> PREDICT
  -> TYPE THE COMMAND / CODE
  -> OBSERVE REAL STATE
  -> BREAK / DEBUG
  -> EXPLAIN THE MECHANISM
  -> REPEAT
  -> HANDLE A HARDER CASE
  -> GENERALIZE TO THE NEXT PROJECT
```

Experience is accumulated repetitions under increasingly realistic pressure, not
the number of frameworks or patterns encountered.

## Fundamental fluency

The following are not side topics. They are part of daily backend work and must
be practiced until the user can operate them naturally.

### Shell and operating-system interaction

Practice repeatedly:

- process and parent/child process thinking;
- current working directory and path resolution;
- environment variables and process inheritance;
- stdin/stdout/stderr and exit status;
- pipes and redirection;
- executable lookup through PATH;
- permissions and executable scripts;
- foreground/background processes;
- Bash quoting, expansion, functions and failure behavior.

The expected reflex is not to remember every flag. It is to know how to inspect
the environment, use `--help`/manuals, predict process behavior and verify it.

### Git

Git must become normal daily state inspection rather than a ceremonial commit
step.

Core reflexes include repeatedly using and understanding:

```text
git status
git diff
git log
git branch
git switch
git add
git commit
git fetch
git merge-base
git rev-list
git show
git restore (only with understood scope)
```

Before mutation, inspect repository state. Understand working tree, index, local
refs, remote-tracking refs and commits as separate state.

Do not memorize recovery commands without first identifying which Git state is
wrong.

### Docker

The target reflex is:

```text
need a containerized dependency
 -> open official image / Docker docs
 -> understand image defaults and required environment
 -> formulate docker/compose command
 -> run
 -> inspect container/process/network/volume state
 -> diagnose from actual output
```

Repeatedly practice:

- image vs container vs process;
- create/start/stop/remove lifecycle;
- environment;
- port publishing;
- network namespaces and service discovery;
- bind mounts and named volumes;
- Compose lifecycle;
- logs, exec, inspect and runtime observation.

Docker Hub/official image documentation should become a normal first source when
using a new image.

### PostgreSQL and command-line database work

Do not treat the database only as something accessed by application code.

Practice operating it:

- connect with `psql`;
- inspect databases, roles, schemas, tables and constraints;
- execute SQL manually;
- inspect transaction effects;
- read SQL errors and SQLSTATE when relevant;
- use PostgreSQL CLI tools such as `createdb`, `dropdb`, `pg_isready`;
- trace PostgreSQL official documentation for concurrency/isolation behavior.

The user should be comfortable moving between Bash, PostgreSQL CLI, SQL and Go
code while knowing which process/system owns each operation.

### Go

Go fluency is built from repeated small implementations, not large pasted files.

Practice:

- declarations, values, pointers and methods;
- errors and wrapping;
- packages/imports/visibility;
- interfaces only when a real contract earns them;
- context and cancellation;
- resource ownership and defer;
- goroutines/channels/synchronization where pressure requires them;
- net/http;
- database/sql/pgx behavior;
- Go test mechanics;
- reading package documentation and source.

When encountering an unfamiliar function, do not immediately ask for a complete
usage example. Trace its signature, contract, errors, lifecycle and implementation
when needed, then write a minimal experiment.

## Research reflex

AI is a research and reasoning amplifier, not a replacement for source literacy.

For a material unknown:

```text
state the question precisely
 -> identify authoritative source
 -> read the smallest relevant section
 -> inspect implementation source if the contract is still unclear
 -> form an own-words model
 -> make a prediction
 -> run a discriminating experiment
 -> reconcile observation with the source
```

Primary-source order:

1. specification / official product documentation;
2. official package/API documentation;
3. implementation source;
4. local experiment;
5. mature OSS usage;
6. secondary articles/tutorials.

The user should increasingly perform the navigation personally: locating docs,
searching symbols, following source references and deciding what evidence is
relevant.

## Difficulty progression

Training should become progressively harsher.

```text
happy path
 -> invalid input
 -> missing dependency
 -> stale state
 -> incorrect configuration
 -> partial failure
 -> process restart
 -> concurrent requests
 -> race / ordering pressure
 -> migration failure
 -> corrupted/dirty local state
 -> operational recovery
 -> multiple actors/dependencies
 -> performance/resource pressure
```

Do not add complexity for entertainment. Add it when it exposes a real failure
mode or engineering trade-off.

The point of a hard case is to force a new observation, invariant or boundary.

## Risk and value thinking

Every increase in complexity must answer:

```text
What risk/value pressure exists?
What is the cost of doing nothing?
What new mechanism reduces that risk or creates that value?
What complexity does the mechanism introduce?
How will we prove that the trade is worth it?
```

Complexity is a cost to exchange for demonstrated operational or business value,
not a marker of seniority.

## Multiple role lenses

The same requirement should periodically be viewed through different roles.

### BA / product lens

- What user/business problem exists?
- Which rule is actually required?
- What facts are known versus assumed?
- Which edge cases change business meaning?

### Fresher developer lens

- Can I identify the files/functions involved?
- Can I understand the syntax and run the basic workflow?
- Can I reproduce a simple failure and read the error?

### Middle developer lens

- Where does responsibility belong?
- Which invariant must be enforced?
- What are concurrency/failure cases?
- What tests provide useful evidence?
- What coupling is being introduced?

### Senior / system lens

- Who owns durable truth?
- Which failure domains exist?
- What can fail independently?
- What must be observable and recoverable?
- Which operational/security/data risks exist?
- Which abstraction/boundary is now worth its cost?
- How does this affect deployment, evolution and other teams?

These are perspectives for reasoning, not titles to role-play mechanically.

## Daily engineering loop

A realistic work session should often include multiple surfaces:

```text
inspect Git
 -> inspect issue/requirement
 -> inspect current source
 -> research unknown mechanism
 -> edit code
 -> compile/test
 -> run dependency
 -> use CLI/database tools
 -> inspect logs/state
 -> reproduce failure
 -> debug
 -> review diff
 -> commit coherent work
 -> explain result and trade-off
```

The goal is to become comfortable moving between these surfaces without treating
each one as a separate school subject.

## Repetition ledger

Important primitives should recur naturally across BDSPro work.

When a command such as `git status`, `docker compose ps`, `psql`,
`go test`, `go doc` or a shell pipeline appears, prefer having the user type
and interpret it repeatedly rather than hiding it behind automation immediately.

Automation becomes valuable after:

1. the primitive is understood;
2. the repeated workflow is clear;
3. human repetition has become a source of error or friction;
4. the automation preserves observability and debuggability.

## AI collaboration contract

Use AI to:

- sharpen the problem statement;
- expose missing assumptions;
- identify relevant official sources/symbols;
- compare options and trade-offs;
- propose experiments;
- review user-written code;
- challenge reasoning;
- explain errors after the user has inspected them;
- help compress learned truth into durable docs.

Do not use AI by default to:

- generate the whole implementation;
- replace documentation/source reading;
- skip compiler/runtime feedback;
- choose architecture before pressure exists;
- turn unfamiliar mechanisms into opaque wrappers;
- produce interview stories disconnected from actual work.

The desired progression is that the user needs less direct implementation help
while using AI more effectively for research, review and complex reasoning.

## Career evidence

BDSPro work should produce truthful career evidence.

For each meaningful milestone, preserve enough information to later explain:

```text
Problem
Why it mattered
Initial system state
Options considered
Mechanism chosen
Code/operational work performed personally
Failure encountered
How it was diagnosed
Evidence that the result works
Trade-offs
What would change at larger scale
```

CV/interview material must be derived from work actually performed and understood,
not architecture vocabulary memorized afterward.

## Acceptance standard

A concept is not considered operationally learned merely because it was explained
once.

Stronger evidence is:

- the user can recognize when the concept is relevant;
- can locate the correct documentation;
- can write/use the primitive with limited prompting;
- can predict common behavior;
- can diagnose at least one failure;
- can explain the mechanism in their own words;
- can identify when *not* to use it;
- can transfer it to a different project/problem.

## Relationship to other continuity documents

- `MINDSET.md`: how architecture evolves from pressure.
- `PRACTICE-PROTOCOL.md`: how an individual coding/research/debugging step is run.
- this document: how repeated work is structured to build broad professional
  engineering fluency over time.
- `CHECKPOINT.md`: where BDSPro is now and what pressure is next.

All four are required to recover the intended working mode.
