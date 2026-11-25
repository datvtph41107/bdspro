# ADR-RESET-001 — Requirement-led engineering governance
Date: 2026-11-25. Status: owner-directed policy reset; implementation architecture unchanged. Baseline: dev@6fbb2104806d45ed1de8005609694acf0d1c7d1f.

## Problem and decision
The owner reports a two-developer startup with one backend developer, limited production/testing experience, unclear business requirements, and no supplied traffic/incident baseline. Earlier distributed design was adopted without a traceable decision history available to the owner. The present documentation still elevates Gitea alignment and earlier technical workstreams. This imposes learning and maintenance obligations without a confirmed business benefit.

Replace the architectural target with an evidence-based, sequential, requirement-led engineering process. Use external source and research as optional comparators. Begin with an agreed business workflow and the actual baseline, select the smallest adequate mechanism and preserve the rationale, limitations and outcome. Prior decisions retain their historical acceptance; they are not new implementation authority. No existing code is approved or rejected by this policy alone.

## Alternatives
- Keep Gitea alignment: recognizable reference, but risks adopting another project's historical constraints and excess complexity.
- Rewrite immediately to a flat/DI architecture: may reduce some mechanisms but repeats the same unvalidated target selection and risks regressions.
- Requirement-led incremental work (selected): learn and verify in context, avoid large unproven changes; requires disciplined evidence and small ongoing records.

## Consequences and verification
Do not resume F40, Gitea alignment or parallel plans without new scoped approval. Retain previous docs; mark entry points historical. First active case is baseline and business requirement discovery. No runtime source, dependency, migration, deployment or test changes are authorized here. No claim that cost, reliability or security has improved. Verify documentation paths and PR scope; owner reviews before merge.
