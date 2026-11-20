# BDSPro — Resume across ChatGPT sessions

**Purpose:** Durable context, not a substitute for verifying GitHub or a request to execute old plans automatically.
**Active phase:** [RG-00](CURRENT_PHASE.md) — research governance and evidence/context continuity only.
**Updated:** 2026-11-20.

## Fast read path for a new conversation

1. Root [AGENTS.md](../../../AGENTS.md) and [this README](README.md).
2. [CURRENT_PHASE.md](CURRENT_PHASE.md) — exact ONE active phase, acceptance and non-goals.
3. [EVIDENCE_LEDGER.md](EVIDENCE_LEDGER.md) — pinned facts and contradiction warnings; recheck SHA.
4. [REQUIREMENTS_AND_EVIDENCE.md](REQUIREMENTS_AND_EVIDENCE.md) and [research protocol](../research-protocol.md).
5. Read *only the source files and case study relevant to the active question*, plus [DESIGN_CASE_TEMPLATE.md](DESIGN_CASE_TEMPLATE.md).
6. Read historical Gitea-alignment or parallel-work material only for a named reason; their old Gin/P01/F40 instructions are NOT current implementation authorization.

## User intent (current)

- Long-lived Go BDSPro monolith; invest in deeply understandable and testable core mechanisms, package ownership, naming, interface contracts, runtime behavior, security, deployment and maintenance—not superficial pattern reproduction.
- Study actual architecture/operations of **Gitea** and **Laravel DRX**, with official Go/net/http/Chi/HTTP/security semantics and other implementations when appropriate.
- Work one phase and one core problem at a time. No verdict, architecture target tree or refactor proposal before tracing its full producers/consumers, import/runtime/operation effects and counterexamples.
- Preserve existing BDSPro `Group/Combo/Mount/typed actions` declaration contract during research; any later alteration needs separate approval.
- User may choose to update local code later **after** understanding and explicitly approving a scoped design; do not infer approval for production code changes from documentation work.
- Prefer specific code citations, SHA/path/line, caller/callee, compile/run examples with exact commands and *real* output, security/failure cases and NÊN/KHÔNG NÊN/CHƯA NÊN only when justified.

## Verified source snapshots (navigation, verify again)

BDSPro `dev` `abe04e46269632ef671a92bab00319b52696e8f5`; Gitea `fbf986d1c7ba1b0ad0078105b3adb0a73b769295`; Laravel DRX `main` `767737d3a9530d0ec3ff94876752e40ccb302a15`. Source citations in [EVIDENCE_LEDGER.md](EVIDENCE_LEDGER.md).

Some documents describe an earlier Gin-based implementation or a multi-branch implementation roadmap; current source showed Chi at that snapshot. Never adopt old state or auto-continue an old PR.

## Current work / blockers

- RG-00: documentation established; **user review pending**. No technical HTTP architecture approved. No code, tests or deployment edits authorized in this phase.
- Architecture of HTTP ingress, source-IP ownership, body/CORS policy, request/response tracking, global logging/recovery, request-scoped dependencies, security admission and package boundaries remains **OPEN**. Previous suggested directory layouts are hypotheses, not choices.
- No production traffic/benchmark/CI result supplied for a redesigned HTTP stack. Do not claim one.
- Existing architecture/governance files may contradict current runtime; preserve them as historical evidence and resolve contradictions with source links before changing docs.

## Handoff update rule at the end of a focused session

Update only facts actually inspected and scope of current phase:

```text
Date + phase + owner:
Task/question:
Source branch + before SHA + after SHA (if any):
Sources inspected (repository @ SHA, file:line, caller/callee):
Facts OBSERVED / inference / unverified:
Cross-subsystem graph examined / NOT INSPECTED:
Decisions: OPEN/PROPOSED/ACCEPTED (+ approver) / REJECTED:
Files changed (docs/code) and actual test commands/outcomes:
Conflicts and risk/security/operational unknowns:
One precise next action in the same phase:
```

Do not append entire old transcripts or overwriting this file with guesses. A new assistant must verify HEAD and reopen exact source before answering. When a phase is approved complete, update CURRENT_PHASE to record closure; promote at most one successor explicitly approved by the user.

## Self-contained startup prompt

"Đang nghiên cứu BDSPro. Hãy đọc `AGENTS.md`, `docs/architecture/research-governance/{CURRENT_PHASE,REQUIREMENTS_AND_EVIDENCE,EVIDENCE_LEDGER,SESSION_HANDOFF}.md`. Xác minh branch/HEAD source thật, phân biệt tài liệu lịch sử và runtime. Chỉ tiếp tục phase đang hoạt động, không refactor hay chốt kiến trúc nếu thiếu requirement + full source/caller-consumer chứng cứ + phản biện + điều kiện nghiệm thu. So sánh Gitea/Laravel DRX khi liên quan; đánh dấu mọi khoảng trống OPEN. Cho biết một bước nghiên cứu có thể tái hiện tiếp theo."
