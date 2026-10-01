
# Engineering Workflow & Command Reflex

## Purpose

This document defines the everyday operating rhythm for BDSPro.

The goal is not to turn every command into a lecture. The goal is to build a repeatable professional reflex:

~~~text
know why the command is needed
 -> understand what state it reads or mutates
 -> understand its syntax/arguments
 -> trace the primary documentation when the mechanism is new
 -> predict the result
 -> type it personally
 -> inspect real output
 -> explain what changed
 -> repeat until fluent
~~~

A developer should eventually move quickly through familiar primitives without losing the ability to reopen the mechanism when a new option, failure mode or environment appears.

## Command-learning lifecycle

### First encounter or materially new use

Use the full trace:

~~~text
1. CURRENT PROBLEM
2. WHY THIS COMMAND / API NOW
3. STATE READ OR MUTATED
4. COMMAND / SIGNATURE ANATOMY
5. PRIMARY DOCUMENTATION
6. PREDICTION
7. USER TYPES IT
8. REAL OUTPUT
9. INTERPRETATION
10. FAILURE / EDGE CASE
11. OWN-WORDS EXPLANATION
12. REPEAT
~~~

### Familiar repeated use

Do not re-teach everything.

Use the shorter reflex:

~~~text
intent -> type -> observe -> interpret -> continue
~~~

If a new flag, new environment, unexpected output or destructive behavior appears, expand back to the full trace.

This is how fluency grows without turning work into permanent tutorial mode.

## Command anatomy

For a CLI command, the user should gradually learn to parse:

~~~text
program
subcommand
options / flags
option values
positional arguments
pathspec / resource selector
shell operators / redirections
environment variables
~~~

Example shape:

~~~text
git diff --stat -- main.go
│   │    │      │  └──── pathspec
│   │    │      └─────── separator before paths
│   │    └────────────── option
│   └─────────────────── subcommand
└─────────────────────── executable
~~~

The exact grammar belongs to the tool documentation. Do not infer a flag from memory when its semantics matter.

## State-first rule

Before using a command, identify the system state it interacts with.

Examples:

~~~text
git status
  reads: HEAD / index / working tree / branch tracking information
  mutates: nothing

git fetch origin
  reads: remote repository
  mutates: local object database + remote-tracking refs / FETCH_HEAD
  does not normally rewrite working files

git add
  reads: working tree
  mutates: index

git commit
  reads: index + metadata
  creates: commit object / advances current branch ref

docker compose ps
  reads: Docker/Compose runtime state

docker compose up
  may create/reconcile/start containers and networks

psql
  starts a PostgreSQL client process
  SQL executed through it may read or mutate database state

go test
  builds test binaries, executes tests and reports test/process results
~~~

Never group all CLI commands under "just commands". Each one participates in a different state machine.

## Risk classes

### R0 — Observation

Examples:

- git status
- git diff
- git log
- docker compose ps
- docker logs
- read-only SQL queries

Normally execute freely and use them often.

### R1 — Local reversible mutation

Examples:

- editing source
- git add
- creating a local branch
- starting/stopping a disposable local process

Understand the changed state and recovery path.

### R2 — Shared/history/runtime mutation

Examples:

- push
- merge
- migration application
- container/volume lifecycle changes
- writes to a shared database

Inspect target and scope first.

### R3 — Destructive or difficult-to-recover mutation

Examples:

- dropping a database
- deleting a Docker volume
- force-pushing
- resetting or cleaning unknown Git work
- production data mutation

Require explicit target identity, blast-radius reasoning and a recovery/backup story before execution.

## Primary-source reflex

### Git

Primary documentation:

- https://git-scm.com/docs/git-status
- https://git-scm.com/docs/git-diff
- https://git-scm.com/docs/git-fetch
- https://git-scm.com/docs/git-branch
- https://git-scm.com/docs/git-add
- https://git-scm.com/docs/git-commit

For a new Git option, first locate it in SYNOPSIS and OPTIONS, then map it onto the working-tree/index/ref model.

Useful local discovery:

~~~text
git <command> -h
git help <command>
~~~

### Bash / shell / Linux

Prefer local/built-in documentation first:

~~~text
help <bash-builtin>
type <name>
command -v <name>
<command> --help
man <command>
~~~

For Bash language semantics, use the GNU Bash Reference Manual.

Always distinguish:

~~~text
shell syntax
vs
external executable behavior
~~~

Primary manual:

- https://www.gnu.org/software/bash/manual/

### Docker

Primary documentation:

- https://docs.docker.com/reference/cli/docker/
- https://docs.docker.com/reference/cli/docker/compose/

For an image, also read the official image documentation and tags before writing the final run/Compose configuration.

Trace:

~~~text
CLI
 -> Docker Engine
 -> image
 -> container configuration
 -> process
 -> network/storage state
~~~

### PostgreSQL / SQL

Primary documentation:

- https://www.postgresql.org/docs/current/app-psql.html
- https://www.postgresql.org/docs/current/app-createdb.html
- https://www.postgresql.org/docs/current/app-dropdb.html
- https://www.postgresql.org/docs/current/app-pg-isready.html
- https://www.postgresql.org/docs/current/sql-commands.html

A PostgreSQL CLI utility is a client, not "the database itself".

For every connection-oriented command, identify:

~~~text
server / host
port
database
PostgreSQL role
credential source
operation privilege
~~~

Before destructive SQL/database operations, prove target identity.

### Go

Primary documentation:

- https://go.dev/doc/
- https://go.dev/doc/cmd
- https://pkg.go.dev/

Local discovery is part of normal work:

~~~text
go help <command>
go doc <package-or-symbol>
go env
go list
~~~

When using an unfamiliar function:

~~~text
call site
 -> signature
 -> documentation contract
 -> source implementation if necessary
 -> minimal experiment
 -> own-words model
 -> production use
~~~

## Git working reflex

### Start or recover a work session

Normally begin by observing:

~~~text
git status
git branch --show-current
git log --oneline --decorate -n <small-number>
~~~

Fetch when current remote knowledge matters:

~~~text
git fetch <remote>
~~~

Then classify:

~~~text
working tree clean/dirty?
staged changes?
untracked files?
local-only commits?
remote-only commits?
diverged?
~~~

Do not choose pull/rebase/reset until this state is understood.

### Before coding

State the work intent.

Ask:

~~~text
Is this a coherent work item?
Does it need isolation from main?
Will it take multiple commits?
Can another stream change main concurrently?
~~~

Only then decide whether a branch creates value.

Branch names should communicate intent, not satisfy a ritual.

Possible vocabularies:

~~~text
feat/...
fix/...
test/...
chore/...
recovery/...
~~~

The prefix is secondary; responsibility is primary.

### During coding

Use git diff repeatedly.

The purpose is not only pre-commit review. It is continuous feedback about what your hands have changed relative to Git state.

Use path-scoped diff when focusing one area.

### Before commit

Ask:

~~~text
What single coherent idea does this commit record?
Which files/hunks belong to it?
What evidence says it works?
Would reverting this commit make conceptual sense?
~~~

Then inspect the staged result, not only the working tree.

The commit message should summarize the work unit honestly.

### End of work

Review:

~~~text
git status
git diff / git diff --cached as relevant
tests / runtime proof
git log
~~~

A session should end with a known state, even when work intentionally remains uncommitted.

## Docker working reflex

When introducing or operating a dependency:

~~~text
1. identify the required service/runtime
2. read official image/tool docs
3. identify image/tag
4. identify required environment
5. identify ports
6. identify storage lifecycle
7. identify network path
8. start it
9. inspect runtime state
10. inspect logs
11. connect/use it
12. stop/recreate it deliberately
13. observe what persists
~~~

Do not automate away the primitive lifecycle until it is understood.

## PostgreSQL working reflex

Move naturally among:

~~~text
shell
 -> PostgreSQL client utility
 -> psql
 -> SQL
 -> application Go code
~~~

Know which layer you are currently using.

For database work:

~~~text
identify target
 -> connect
 -> inspect
 -> make prediction
 -> execute SQL
 -> query durable state
 -> interpret errors/transaction effects
~~~

Do not treat an application HTTP response as database proof.

## Go coding reflex

For a meaningful function:

~~~text
responsibility
 -> inputs
 -> outputs
 -> errors
 -> side effects
 -> resource ownership
 -> invariant
 -> smallest implementation
 -> compile/test
 -> inspect failure
 -> revise
~~~

Do not start with the final abstraction.

Prefer hand-written small implementations until repetition or pressure earns a helper, interface, package or stronger boundary.

## Test-writing reflex

Before code:

~~~text
requirement
 -> invariant
 -> scenario
 -> given
 -> when
 -> then
 -> observation point
~~~

The user writes the test name and body.

Do not count a copied green test as learning evidence.

## AI response contract

For a new or important command/API, the assistant response should normally expose:

~~~text
WHY NOW
COMMAND / API SHAPE
WHAT EACH MATERIAL ARGUMENT MEANS
STATE READ / MUTATED
PRIMARY SOURCE TO READ
WHAT TO PREDICT
WHAT THE USER SHOULD TYPE
WHAT OUTPUT TO OBSERVE
WHAT CONCLUSION IS / IS NOT JUSTIFIED
~~~

For familiar repeated commands, compress this heavily so real work retains pace.

When the user supplies output, start from that output rather than replacing the exercise with an idealized solution.

## Daily work loop

A realistic BDSPro session should trend toward:

~~~text
OBSERVE REPOSITORY STATE
 -> RECOVER CURRENT REQUIREMENT/PRESSURE
 -> RESEARCH THE UNKNOWN
 -> PREDICT
 -> TYPE CODE/COMMAND
 -> BUILD/TEST/RUN
 -> INSPECT PROCESS/DB/LOG/GIT STATE
 -> DEBUG
 -> REVIEW DIFF
 -> COMMIT COHERENT EVIDENCE
 -> UPDATE DURABLE CHECKPOINT WHEN MATERIAL
~~~

This is deliberately close to normal engineering work.

## Progression standard

The target is not permanent slowness.

~~~text
explicit reasoning
 -> repeated deliberate practice
 -> pattern recognition
 -> faster execution
 -> automatic basic reflexes
 -> attention moves to harder system problems
~~~

When the basics become automatic, do not artificially slow them down. Preserve the ability to explain and debug them, then spend cognitive effort on the next pressure.
