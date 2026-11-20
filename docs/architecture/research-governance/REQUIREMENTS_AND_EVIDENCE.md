# Requirement engineering and evidence standard

**Status:** Methodological contract, not approval of a technical design.

## 1. Requirements are discovered, not copied from folder trees

Record separately:

- **User goal** (explicit): e.g. maintain a long-lived Go monolith, explain execution, keep present Group/Combo/typed action declaration while under review, support Linux binary and Docker.
- **Observed current mechanism**: exact function/commit/call graph; no verdict.
- **Engineering force**: reproducible complexity/failure/change pressure, or an invariant implied by protocol/language/operational responsibility. Mark hypothetical when it has not manifested.
- **Functional/system property**: what HTTP, resource, actor, worker, DB or external consumer must accomplish.
- **Quality requirement**: correctness, debuggability, security, maintainability, change safety, latency, concurrency, availability; define observable measure and environment.
- **Non-goal**: work intentionally deferred to avoid creating infrastructure by speculation.

A proposed package hierarchy is an **output of a tested ownership model**, never the input requirement. Before selecting a target tree, inspect all producers and consumers, imports and runtime dependencies, initialization/shutdown, configuration, errors, tests, current deployment and change surface.

## 2. Investigate one mechanism end-to-end

Mandatory trace for each central operation:

`entrypoint → constructor/init (once) → registration/composition → HTTP/worker trigger (per event) → middleware chain → handler → service/DB/external calls → response/error → cleanup/shutdown`.

For every edge include file, function, import or reference, exact SHA and line range, data/state passed, cancellation, ownership, error/early-return behavior and execution frequency. For a security boundary include trusted/untrusted inputs, attack assumptions, failure response and log/trace visibility.

Distinguish declaration-time function invocation, closure construction, request-time invocation and deferred execution. If any edge cannot be inspected mark `TRACE GAP`. Do not infer an entire system from one middleware.

## 3. Evidence hierarchy and falsifiability

| Evidence class | Required attribution | What it can establish | What it cannot |
| --- | --- | --- | --- |
| VERIFIED SOURCE FACT | repo SHA/path/lines, caller and consumer | Actual inspected implementation/control flow | Why author chose it; production outcomes |
| OFFICIAL CONTRACT | versioned Go/RFC/Chi/DB docs | Normative semantics/specified API | BDSPro conforms |
| CONTROLLED TEST | command, env/version, test file, actual output | Behavior for chosen input | Production safety or universal superiority |
| REPRODUCIBLE MEASUREMENT | env/load/baseline/units, tool, result | Cost in measured scenario | Cost in other environments |
| UPSTREAM DESIGN INTENT | official design issue/PR/document, matching revision | Author rationale if explicit | BDSPro must copy it |
| INFERENCE / HYPOTHESIS | clearly labeled assumptions + counterexample | Plausible design interpretation | Proven fact |

A link alone is not analysis. Explain what lines prove, why the mechanism leads to the behavior, which edge remains unknown, and the case that might falsify the proposed conclusion. Unverified absence claims must state the search scope and cannot become global assertions.

## 4. Comparative study: Gitea × Laravel DRX × BDSPro

1. Define an independent problem from Go/HTTP/security/database semantics.
2. Trace Gitea at a pinned revision, with registration, execution and real consumers; check upstream design rationale only if available.
3. Trace Laravel DRX at a pinned revision, its provider/kernel/middleware/route/service/operational boundaries; explain which behavior is Laravel-framework-specific and cannot transfer directly.
4. Trace current BDSPro and other callers (Web/CLI/worker/DB), not just a matching name.
5. Compare `net/http` or Chi minimal baseline, current design, Gitea style and DRX-inspired organizational alternative on the SAME scenario.
6. Document **what to adopt / not adopt / defer** only after evidence and constraints; compare costs and negative cases, not just syntax or directory similarity.

Use a different reference only when it answers a specific unresolved mechanism. If a source is inaccessible, say so and leave evidence incomplete.

## 5. Mandatory impact matrix before architecture proposal

For each contemplated core boundary, record `INSPECTED`, `NOT INSPECTED`, or `N/A (reason)` for: application startup; HTTP APIs/route matching; CLI and migrations; workers/queues/cron; auth and authorization; DB transactions; Redis and shared clients; public contract and error mapping; concurrency/cancellation; Linux binary and Docker/proxy; logging/tracing/audit; security/privacy; testing and CI; import/package and consumer inventory; feature expansion/removal.

A file-tree target remains a `HYPOTHESIS` until these producer/consumer edges and imports are grounded.

## 6. Security gate for all ingress/control-plane work

Specify assets, trust boundaries, threat actor capability, attack preconditions and actual ingress topology. Separate transport and reverse proxy, global HTTP middleware, route authentication/authorization, operation admission and domain/transaction checks. Treat X-Forwarded-For and Origin as untrusted until their specific trust policy applies. CORS is not authentication; rate limit is not business quota or guaranteed bot defense.

Include normal/direct-client, proxy, malformed header/body, preflight, anonymous, unauthorized, concurrent, timeout, Redis unavailable, panic before/after response commit, route miss and operational logging scenarios **when relevant**. Minimize sensitive logs and avoid unbounded metric labels; assess fail-open vs fail-closed per dependency. Security controls and monitoring are never asserted "optimal" without an explicit risk/cost model.

## 7. Decision and change-cost gate

For each option (including keep-as-is and a smaller direct implementation), report:

- API and import changes, exact callers/consumers, constructor and runtime resource ownership.
- Invariant preservation, positive/negative cases, new failure modes and security tradeoffs.
- Code surface (files/functions) observed, coupling, test setup, operational overhead, knowledge burden, migration/revert.
- Measured latency/allocations/Redis ops/log cost if available; otherwise `NOT MEASURED`.
- Status `OPEN → PROPOSED → ACCEPTED` only with reviewer/user approval; architectural recommendation is **not** acceptance.

## 8. Required output for every single-phase investigation

`Question → requirements → source inventory/pinned SHAs → actual code with lines → call graph and invocation timing → cross-subsystem map → adversarial test → source contrasts → alternative code shapes → costs → uncertainties → explicit STATUS → reproducible next evidence task`.

If evidence is insufficient, finish with a narrower research task, not a verdict or preapproved target tree.
