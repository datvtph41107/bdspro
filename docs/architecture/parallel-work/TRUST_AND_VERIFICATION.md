# Trust Engineering Protocol: Verification, Context, Guardrails

The reference is Lauren Tan's **trust ladder** discussion with Matt Pocock ([record/transcript](https://mwolf.dev/library/live-poteto-creator-of-pstack-on-shipping-1-000-s-of-pr-s-a-month-at-spacex/live-poteto-creator-of-pstack-on-shipping-1-000-s-of-pr-s-a-month-at-spacex)). Her emphasis is on verification, creating an environment where correct code is easier to produce, extracting deterministic review steps into tools, and increasing autonomy **only after evidence**. This document is **our BDSPro adaptation**, not a claim that Lauren prescribed these exact BDSPro roles, gates or measurements.

## Three engineering pillars

**1. Verification / evals:** every worker has a concrete input task, observable output (diff/tests/CI), and falsifiable acceptance criteria. When a regression recurs, create a test, static check, lint rule, or repeatable script; a reviewer should not need to repeatedly remember the same rule.

**2. Context engineering:** a new chat starts from a short canonical read order, fixed branch/commit and task scope, pinned Gitea source and local call sites. It writes decisions and handoff evidence to a PR, not to inaccessible chat memory. Summaries are navigation aids, not authoritative source.

**3. Guardrails:** write-set enforcement, integration owner, locked version/stack, contract gates, representative tests, adversarial verification and a revert plan. No blind auto-merge, especially for Auth, transaction authority, migrations or security-critical changes.

## BDSPro trust ladder

| Level | Allowed autonomy | Minimum repeatable evidence |
|---|---|---|
| T0 — inspect | read/research and propose plan only | exact source links; identify owners and contract |
| T1 — scoped implementation | commit to own branch; **draft PR** | no unapproved paths, gofmt, focused tests, source-to-Gitea diff |
| T2 — tested implementation | request review, coordinator may integrate | final HEAD full CI green, runtime/relevant security tests, independent review and handoff |
| T3 — routine changes | quicker review for **proven low-risk lane-specific changes**, no bypass of controls | repeated accepted PRs with no escaped regressions, executable guards, rollback process |
| T4 — delegated merge (future only) | maybe automation for genuinely reversible, noncritical files | explicit human policy change, branch protections, veto/rollback, measured history |

**Current policy: T1 for new worker chats.** Passing one earlier commit's CI does not certify current HEAD. No autonomous merge into `dev`. P01 draft PR #218 and new lanes must remain independently auditable.

## Evidence hierarchy and mandatory self-review

For each claim of completion attach:
- exact base SHA, head SHA and changed file paths;
- Gitea source permalink at a pinned commit, relevant import/caller and semantic comparison;
- BDSPro actual diff and which invariants are preserved;
- full final-HEAD CI run/job status, tests run, observed failures and the corrections;
- security-sensitive or integration-sensitive manual checks and untested risks;
- a reviewer-authored **counterexample** that tries to break the design (invalid handler, expired session, nested transaction, Redis failure, etc.);
- PR link and outstanding blockers; a handoff in the worker's own document.

Never report `go test ./...` as successful if only a subset passed; never report "CI green" from a parent commit or a run still queued.

## Determinism before agent judgment

Tasks that can be made deterministic should be: gofmt, package-boundary checks, `go test`, SQLC generated diff checks, OpenAPI route inventory, build, migration fresh-schema validation, static grep for forbidden imports, runtime HTTP smoke, and exact CI artifact checks. Human/agent reasoning should focus on **semantics, risks, tradeoffs and source ownership**, not remembering a convention.

## Metrics, not agent-count theater

Track per lane (per merged PR): first-pass CI rate, number of failed CI runs and root causes, files changed outside ownership, cross-lane conflict count, test coverage of changed behavior, reviewer rejection/rework, escaped regression/revert count, and elapsed integration wait. Consider adding a fourth producer only when three lanes maintain acceptable quality and few boundary conflicts.

There is no universal "95% trust" number; define locally from observed evidence. Throughput (#commits/PRs) is not a substitute for correctness, recoverability and stable interfaces.

## Reviewer checklist

1. Does the code use Gitea's patterns for a **stated runtime reason**, not just similarity?
2. Does the route/model/service ownership match the approved graph?
3. What would break if two sessions applied changes in the opposite order?
4. Can a new chat reproduce the tests and identify the exact current HEAD?
5. Are retries, timeouts, cancellation, transaction rollback, session authority, and secrets safe?
6. Is a rollback possible without corrupting migration state?
7. Was the complete PR tested **after** all commits?
8. Is the report honest about remaining mismatches?

## Relevant sources

- [Lauren Tan × Matt Pocock transcript](https://mwolf.dev/library/live-poteto-creator-of-pstack-on-shipping-1-000-s-of-pr-s-a-month-at-spacex/live-poteto-creator-of-pstack-on-shipping-1-000-s-of-pr-s-a-month-at-spacex)
- [Analysis of trust-before-parallel](https://redreamality.com/blog/lauren-tan-poteto-trust-before-parallel/)
- [Gitea repository](https://github.com/go-gitea/gitea)
- [BDSPro CI and historical P01 PR](https://github.com/datvtph41107/bdspro/pull/218)
