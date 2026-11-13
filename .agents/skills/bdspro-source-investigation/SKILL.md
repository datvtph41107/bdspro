---
name: bdspro-source-investigation
description: Use for evidence-led research of an architecture, runtime behavior, code path, external implementation or claim before proposing BDSPro changes. Do not use as permission to write code.
---

# Investigate a technical claim

1. Read root `AGENTS.md`, `docs/research/PROGRAM_01.md`, `docs/research/PRACTICAL_REASONING.md`, `docs/research/VISUAL_EVIDENCE.md` and any question ID. Check the current BDSPro `dev` SHA; do not rely on an older task board or previous chat.
2. State the practical problem, affected actor/domain, hypothesis and **at least one competing explanation**. Distinguish observed behavior, source comment, inference and experiment.
3. Traverse source **entrypoint → callers → functions/types → data/state changes → error/failure paths → tests → operational dependencies**. Record exact GitHub permalinks pinned to commits. For third-party systems, trace adjacent callers, not filenames alone.
4. Read external design/research, maintainer discussions, PRs and opposing perspectives. Identify authors, version, experimental setup and evidence limits.
5. Build a minimal call/transaction/sequence graph **only when it clarifies a claim**. Mark inferred links; a diagram is not a runtime measurement.
6. Define an adversarial or reproducible check: what result would disprove the hypothesis? Do **not** claim to have run it if only proposed.
7. Produce a brief research handoff: claim; competing claim; source pins; evidence class; mechanism; assumptions; risks; test proposed/run; confidence and next question.

For BDSPro, do not import Discourse's Ruby service DSL, Gitea's Chi/XORM or a proposed pattern without a concrete local need. This skill is read-only until a separate implementation scope is authorized.
