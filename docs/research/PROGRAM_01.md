# Program 01 — Evidence-led Software Engineering Research

**Status:** proposed research charter for review, 2026-11-13. **Owner:** repository governance / coordinator (lane 0). This document is not a product architecture decision or permission to modify application code.

## Purpose

Study **how engineers, researchers, maintainers, companies and communities discover software-development problems, construct arguments, test hypotheses, debate alternatives and turn findings into working systems, tools, methods and products**.

Scope spans computer science, backend engineering, programming and code organization, architecture, AI, testing, product/domain evolution, team work, delivery, security, operations and maintenance. Study a project's **entire engineering life**, including failures and unresolved decisions. Go and BDSPro are useful places for hands-on validation, **not** limits on which disciplines or sources may be studied.

**Do not start by choosing a favorite architecture, framework or repository.** Derive questions from practice and evidence, then examine source and operational implications. The later 3×3 learning matrix and selection of five distinct Go monoliths are **subsequent programs**, not outputs or assumptions of Program 01.

## Research loop

1. **Observe / frame:** describe the actual work and concrete problem. Who encounters it? In which project stage, team and workload?
2. **Question / claim:** write a falsifiable question; separate reported observation, proposed mechanism, causal hypothesis and personal opinion.
3. **Locate sources:** start with primary research, company technical reports, design proposals, engineering case studies, incident postmortems, real code/commits, PR reviews and community counterarguments. Follow citations to predecessors and critics.
4. **Reconstruct method:** identify population, timeframe, dataset, comparison group, experimental or case-study method, assumptions, measurements, costs and conflicts of interest.
5. **Challenge:** look for counterexamples, alternative causal explanations, non-generalizable conditions and the best argument against the preferred conclusion.
6. **Connect to mechanisms:** trace abstract ideas to APIs, ownership, tests, release procedures, failure paths, debugging, business semantics and operations. Pin source code at an exact commit when used.
7. **Evaluate / reproduce:** distinguish observed evidence from proposed tests. Run a bounded experiment on BDSPro or a disposable example only when specifically scoped; do not infer production validity from a toy benchmark.
8. **Record / decide:** state what is known, uncertain, rejected or deferred; the next question; and what could change the assessment. No implementation/merge without existing BDSPro controls.

The goal is **learning how evidence is produced and put to work**, not accumulating links or endorsements.

## Practical reasoning and visible technical evidence

The research method also starts from first-hand **problem discovery**, not just reading papers: ask why a mechanism exists, who it helps, what it costs to change, and what failure teaches. Combine **first-principles analysis** of protocols/source/state with **contextual systems thinking** about users, domains, team and long-term operations. For recurring inquiries use **Surface → Root → Mechanism → Impact → Options/Solution → Operation** and preserve the strongest dissenting hypothesis.

Read [PRACTICAL_REASONING.md](./PRACTICAL_REASONING.md) for reasoning based on realistic scenarios, falsifiable arguments, and a smallest-action loop. Read [VISUAL_EVIDENCE.md](./VISUAL_EVIDENCE.md) to distinguish explanatory diagrams from observed images, profiles, traces, measured charts and cross-disciplinary photographs. Every visual that supports a claim needs traceable provenance, context, units and limitations. Illustrative numbers in working notes must not be upgraded to actual project evidence.

The four user-provided working texts that informed these lenses are not republished to this public repository; these two pages provide a privacy-conscious synthesized handoff.

## Core research dimensions

- **Inquiry and research method:** empirical software engineering, field observations, qualitative interviews, surveys, controlled experiments, benchmarks, systematic reviews, reproducibility, evaluation bias.
- **Daily engineering:** code structure, naming, patterns, interfaces, review, version control, debugging, testing, refactoring, documentation, developer tools and technical debt.
- **Software mechanics:** HTTP, concurrency, transactions, persistence, security, distributed boundaries, availability, performance, failure recovery and lifecycle.
- **Product / business:** requirements, domain rules, correctness, changing customer needs, cost, incentives, backward compatibility, support and life-cycle constraints.
- **People / organizations:** ownership, coordination, cognitive load, learning, hiring/onboarding, governance, productivity, delivery and human/AI collaboration.
- **Operations and evolution:** observability, incident response, rollback, SRE, migrations, postmortems, deprecation and long-term survivability.

These are **lenses**, not a mandatory folder-based architecture.

## Evidence handling

Every research note must label the type of evidence: **controlled experiment, observational study, survey, field case, source/PR trace, incident report, technical proposal, synthesis/review or opinion**. Record limitations. A survey describes respondents and cannot by itself establish causality. One company's architecture is a context-bound case, not a universal rule. A repository's popularity is not evidence of technical superiority. A performance benchmark must specify environment, workload, baseline and measurement method.

Prefer original materials over summaries. Link the exact section, publication date/version, retrieval date, author/institution, code SHA and relevant discussions if available. Flag a source as **candidate**, **abstract/metadata checked**, **full text reviewed**, **independently cross-checked**, or **replicated**; never silently promote its evidence status.

## Research-note contract (one note per issue)

- **ID / question / owner / date / state:** proposed, researching, disputed, tested, accepted-for-context, rejected, deferred.
- **Problem in practice:** actors, affected work, system stage, business requirement and failure or cost.
- **Claims & competing hypotheses:** strongest argument on each side; which observations would refute them?
- **Sources:** exact URLs + author/date/version; source type and review status; linked studies and contrasting evidence.
- **Method & outcomes:** sample/data, comparison or control, measurements, reproducibility, important missing evidence.
- **Engineering mechanism:** code paths, signatures, dependency, tests, failure behavior, operations and organizational process.
- **Applicability:** what transfers to Go/BDSPro; what is contingent; avoid architecture choices without a local requirement.
- **Decision & next step:** confidence/uncertainty, proposed smallest experiment, owner, next discussion question.

For engineering experiments additionally record repository/ref/SHA, commands, observed outputs, unsuccessful cases, CI status, rollback and untested risks. Never state that unrun tests passed.

## Cross-chat and repository continuity

- This repository's reviewed files are the **durable source of truth** for the program: this charter, [PRACTICAL_REASONING.md](./PRACTICAL_REASONING.md), [VISUAL_EVIDENCE.md](./VISUAL_EVIDENCE.md), [SOURCE_REGISTRY.md](./SOURCE_REGISTRY.md) and [DISCUSSION_BACKLOG.md](./DISCUSSION_BACKLOG.md), plus individual research notes added when investigations actually occur.
- `AGENTS.md` links agents to this program. An agent entering a new chat should fetch current `dev`, read the program and current note/question, and report evidence gaps before suggesting actions.
- ChatGPT account Memory can help recall the *orientation*, but it is not a complete or guaranteed cross-chat record. Do not claim this file has been automatically saved to account Memory.
- **Suggested new-chat starter:** “Trong repo https://github.com/datvtph41107/bdspro, đọc AGENTS.md và docs/research/{PROGRAM_01,PRACTICAL_REASONING,VISUAL_EVIDENCE,SOURCE_REGISTRY,DISCUSSION_BACKLOG}.md từ dev mới nhất. Tiếp tục nghiên cứu câu hỏi [ID], phân biệt bằng chứng với giả thuyết, truy nguyên nguồn gốc và phản biện, không sửa code hoặc merge khi chưa có scope.”
- Repository code edits still follow the separate `docs/architecture/parallel-work/` ownership, branch, lease, tests and approval policy. This program does not supersede those rules.

## First milestone and definition of done

For the initial phase: maintain a **small, diverse, verified seed bibliography**; build a connected set of well-framed questions; read at least one source deeply per selected question, seeking contradicting evidence and tracing its method. A question is “research-ready” when sources, assumptions, outcome measures and next validation step are written down. A reference list alone is **not** an accepted conclusion.
