
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

## Option and flag discipline

Every material flag or argument must earn its presence.

When a command is first introduced, distinguish the base command from the added
modifiers:

```text
base command
  -> default behavior

+ flag / option
  -> what behavior changes?
  -> what problem does that change solve here?
  -> what would happen without it?
```

Examples:

```text
git status
vs
git status --short --branch

The second form is not "more correct". It trades richer human-readable default
output for compact machine/scan-friendly status plus branch information. Use it
when compact recovery/audit output creates value; use ordinary `git status` when
its explanatory output is more useful.
```

Likewise, a Docker `run` command should not accumulate `-d`, `-p`, `-e`,
`--name`, `--network` or volume flags merely because a previous example used
them. For each flag, identify the runtime behavior it changes and whether the
current scenario needs that behavior.

For unfamiliar flags, read the exact option definition in the primary manual
before relying on it.

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

## Repository Core Contract — local ↔ GitHub (active 2026-10-10)

This is the **single operational authority** for BDSPro source-control behavior. It is not an extra governance workflow. Follow `MINDSET.md` for why, `PRACTICE-PROTOCOL.md` for who writes code, `PROPORTIONAL-ENGINEERING-STANDARD.md` for how much rigor, and this section for exact local/remote state.

**Verified remote snapshot 2026-10-10:** GitHub `datvtph41107/bdspro` has **one active remote branch `dev`**, which is also the default. `main` and prior architecture/checkpoint/implementation branches have been removed. PR #1 had been merged as documentation; PR #2 and #3 were closed without code merge. All branch inventories in older checkpoints are **historical**, not a request to recreate them. Later source/branch states must be read live, not guessed from this paragraph.

### Five different states — never collapse them

| Layer | What it means / who changes it | How to inspect |
| --- | --- | --- |
| Local working tree | User's saved file edits, new files and deletions. Can exist without a commit; GitHub cannot see these. | `git status --short --branch`, `git diff` |
| Local index (staging) | Exact intended contents for next local commit. | `git diff --cached` |
| Local `dev` HEAD | Last committed snapshot **on the user's machine**. Not necessarily remote. | `git log -1 --oneline`, `git branch --show-current` |
| Remote-tracking `origin/dev` | The most recently **fetched** snapshot of the remote, not automatically current until fetch. | `git fetch origin --prune`, `git rev-list --left-right --count dev...origin/dev` |
| GitHub remote `dev` | Shared published code/docs at an exact commit. Push updates it; remote tools cannot mutate the user's local workspace. | GitHub commit SHA / fetched branch HEAD |

**PostgreSQL is a sixth independent state**: database schema, rows and `schema_migrations` can differ from **all** Git versions. `migrate up` changes the target database, not Git. Editing/deleting migration files changes local source, not an already-migrated database. Generated `db/sqlc` code is another representation; verify generation and queries when schema changes.

### Before a coding session — short, actual commands

```bash
git branch --show-current
git status --short --branch
git log -1 --oneline
# fetch only when remote comparison/sync matters:
git fetch origin --prune
git rev-list --left-right --count dev...origin/dev
```

For `rev-list --left-right --count dev...origin/dev`, first number = commits local-only, second number = commits remote-only. Zero/zero says committed histories match, **not** that working tree is clean or local PostgreSQL matches. `git fetch` updates remote-tracking refs, **not** checked-out source files. Avoid running `git pull` as a state-discovery command.

Before **staging**, inspect work: `git diff`, `git status --short`. After staging, review **exact bytes planned for commit**: `git diff --cached` (and `git diff --cached --stat`). A coherent commit captures one understandable source decision with a useful validation result. A partial experiment can stay uncommitted in a disposable lab; don't force a commit for every SQL statement or a checkpoint for every terminal command.

### Synchronization: explicit decisions, not blind commands

| Observed local/remote state | Default responsible action |
| --- | --- |
| Working tree clean, `0 0` | No sync necessary. Work locally. |
| Working tree clean, `0 N` | Review upstream diff; `git merge --ff-only origin/dev` is a simple fast-forward. |
| Uncommitted edits/deletions, upstream ahead | Preserve first (scoped stash, reviewed local commit or copy); inspect overlapping paths; then fast-forward/reconcile and restore saved edits. |
| Local ahead, remote `0` behind | Review/tests/status; push with `git push origin dev` when the result should become shared. |
| Both sides have commits | **Divergence**: review commits and paths, choose deliberate integration. No automatic reset/force-push/rebase of shared history. |
| Database was reset but files still tracked | Verify `current_database()` and migration ledger; inspect deletions. Never infer successful app integration from an empty database. |

**Specific current user-reported state (not remotely verifiable):** local `dev` had ten tracked Listing migrations marked `D`; local was 54 remote commits behind at the time of the user's output. The remote `dev` later advanced with documentation. **Do not run a blind pull over the local deletions.** One low-risk approach *if the deleted paths do not conflict with upstream* is:

```bash
git stash push -m "preserve-intentional-migration-deletions" -- migrations
git fetch origin --prune
git merge --ff-only origin/dev
git stash apply
git status --short --branch
git diff -- migrations
```

Check the real output after **each mutation** and stop if Git reports conflict or a refusal; do not proceed mechanically. Keep the stash until deletions are safely represented in a reviewed local commit; `git stash apply` does not discard it. This workflow preserves the migration-file intent but **does not** rebuild/run the application; existing Listing Go/sqlc source still depends on old tables. If untracked migrations were created locally, review/backup those separately first.

### When work has actually finished

A meaningful unit has:
1. Known problem and minimal changed files.
2. User-authored code/SQL, not pasted solution by default.
3. Evidence **appropriate to risk**: e.g., schema experiment output, targeted PostgreSQL test, Go test, runtime query. No automatic giant test matrix.
4. Reviewed staged diff; commit message reflects what changed, not what was merely planned.
5. Pushed to `origin/dev` **when sharing/syncing is needed**, with a known final local status. Local commit without push is legitimate unfinished/personal work; no claims of remote update.
6. Updated `CHECKPOINT.md` / `DECISIONS.md` **only on a material boundary** (new invariant accepted, migration changed, test proof or deployment gate, important failure, work handoff), ideally in the same coherent documentation change or a clearly attributed follow-up. Do not duplicate the entire conversation or rewrite all continuity files every time.

**Authority order:** current observed local working tree + actual PostgreSQL output for local state; current GitHub SHA for shared published history; reproducible tests; focused checkpoint; historical docs; recollection. A stale checkpoint never authorizes overriding live code.

### One branch now; isolate only for real pressure

`dev` is currently the only remote branch, including default. Do not recreate `main`, PRs or CI merely for formality. The user may choose to introduce a stable/release branch later when there is a version worth stabilizing, or a short-lived branch/worktree for a risky parallel experiment. Such a branch needs explicit purpose, end condition and benefit over working directly on `dev`. Work in `bdspro_test` or disposable PostgreSQL schemas for dangerous SQL experiments **without requiring a Git branch**.

### Destructive commands and boundaries

Before `git reset --hard`, `git clean -fd`, `git push --force`, `migrate down`, `dropdb`, `DROP TABLE`, branch deletion or schema-history rewriting: name the actual target, affected data, recovery method and why a non-destructive alternative is insufficient. A user reset of a disposable local database **does not** authorize resetting an unknown remote/shared database. Do not use migration `force` as schema repair; it edits migration version metadata without making DDL match.

**Assistant contract:** users write their SQL/Go/CI and execute operations; assistant explains, gives small exercises, reviews code/output and may edit operating documentation when asked. No automatic implementation, CI, branch creation, Git merging or destructive changes in their name unless they explicitly request that action and its scope.


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
