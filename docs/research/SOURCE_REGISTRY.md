# Program 01 — Initial Source Registry

**Seed status (2026-11-13):** publisher/landing pages checked; this is a **discovery bibliography**, not a claim that full papers, datasets or downstream source code have been independently audited. Links are original publishers, first-party authors or repository hosts wherever possible. Add contrary sources before treating conclusions as established.

| ID | Source / direct link | Evidence type | Primary research use | Critical reading / limit |
|---|---|---|---|---|
| S01 | [Kitchenham, Dybå & Jørgensen, Evidence-Based Software Engineering (ICSE 2004)](https://ebse.webspace.durham.ac.uk/ebse-bibliography/evidence-based-software-engineering/) | Research-method paper | How findings from studies inform engineering decisions | Skill, context and lifecycle bias complicate generalization |
| S02 | [Microsoft Research, The SPACE of Developer Productivity (2021)](https://www.microsoft.com/en-us/research/publication/the-space-of-developer-productivity-theres-more-to-it-than-you-think/) | Framework / peer-reviewed publication | What to measure in engineers' work | Framework is not a causal experiment or a single productivity score |
| S03 | [Google Research, Modern Code Review: A Case Study at Google (2018)](https://research.google/pubs/modern-code-review-a-case-study-at-google/) | Industrial field study (interviews, survey, review logs) | Why code review exists, what it accomplishes and where it hurts | One company, process and historical dataset |
| S04 | [Software Engineering at Google (2020)](https://research.google/pubs/software-engineering-at-google/) | Practitioner book / industrial synthesis | Time, scale, tradeoffs, collaboration, testing and code health | Practices are contextual, not automatically optimal for small Go teams |
| S05 | [Go Proposal Process](https://github.com/golang/proposal) | First-party public governance + design history | How engineering claims become proposals, design docs, decisions and code | Trace individual proposals/issues and implementation for specific claims |
| S06 | [Go Developer Survey 2025 results](https://go.dev/blog/survey2025) | Descriptive developer survey | Pain points with idioms, modules, tooling, AI and daily work | 5,379 cleaned responses; sampling, self-report and selection bias |
| S07 | [DORA 2025 State of AI-assisted Software Development](https://dora.dev/research/2025/dora-report/) | Cross-organization survey + qualitative research | Interaction of AI, engineering process, delivery and organizational system | Correlations and model assumptions are not universal causal claims; see [errata](https://dora.dev/research/2025/errata/) |
| S08 | [CMU SEI: Architecture Tradeoff Analysis Method (1998)](https://www.sei.cmu.edu/library/the-architecture-tradeoff-analysis-method/) | Technical report / architecture evaluation method | Compare quality attributes against business drivers and risk scenarios | Requires concrete context and stakeholder scenarios |
| S09 | [Google SRE: Postmortem Culture](https://sre.google/sre-book/postmortem-culture/) | Industrial operating practice | How incidents create shared technical learning and follow-up actions | Operational experience and guidance, not an RCT |
| S10 | [METR: AI impact on experienced open-source developers (2025)](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/) | Randomized controlled field experiment | Difference between anticipated and measured AI coding productivity | Early-2025 AI tools, 16 developers and selected tasks: avoid extrapolating to every developer or newer tool |
| S11 | [METR reproducibility data / regression](https://github.com/METR/Measuring-Early-2025-AI-on-Exp-OSS-Devs) | Open analysis artifacts for S10 | Check calculations and replicability rather than trusting a summary | Re-running analysis is not replicating the entire field experiment |
| S12 | [GitHub Octoverse 2025](https://github.blog/news-insights/octoverse/octoverse-a-new-developer-joins-github-every-second-as-ai-leads-typescript-to-1/) | Platform activity report | Changes in developer ecosystem, languages and AI adoption | GitHub platform activity != complete global developer population |
| S13 | [Simon Peyton Jones, How to write a great research paper](https://www.microsoft.com/en-us/research/academic-program/write-great-research-paper/) | Research communication talk | How researchers make claims, organize evidence and communicate problems | Advice on presentation, not direct proof of engineering outcomes |

## A linked-reading route, not a reading order imposed on everyone

1. **Producing and evaluating knowledge:** S01 → S13 → S08; compare empirical validation with scenario/tradeoff evaluation.
2. **Daily work and collaboration:** S03 ↔ S02 ↔ S04; inspect what is measured versus what practitioners say they value.
3. **Open-source decision histories:** S05 → select a concrete Go design proposal → discussions/rejected alternatives → merged source and tests.
4. **Observability and business impact:** S09 ↔ S07; connect incidents, delivery constraints, postmortem actions and research on organizations.
5. **AI/software-development tensions:** S10 ↔ S11 ↔ S07 ↔ S06 ↔ S12; compare controlled trial, descriptive self-report, organizational research and ecosystem telemetry.

## Review protocol

For each source, record: published version/date, checked date, authors/institution/funding, population/data, method, precise claim, limitations, contrary evidence, applicability and reproduction status. Preserve **source-to-claim links** in a per-question note, not just a bibliography entry. Add sources outside Go and outside large tech companies when they test a stronger competing hypothesis.

**Source selection principle:** prefer diverse **methods and disagreement** over volume, popularity or prestige.
