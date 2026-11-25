# Documentation reset baseline — 2026-11-25
Repository: datvtph41107/bdspro, dev@6fbb2104806d45ed1de8005609694acf0d1c7d1f. Scope: documentation/agent policy only.
- [AGENTS.md](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/AGENTS.md): earlier canonical process settings and historical yet labeled active F40; overlapping directions, not a runtime observation.
- [Old skill](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/.agents/skills/architecture-research/SKILL.md): mandatory Gitea trace before BDSPro; replaced with requirement-led investigation.
- [Old alignment guide](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/docs/architecture/gitea-alignment/README.md): instructs keeping Gin; [go.mod](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/go.mod) declares Chi and Asynq. Dependencies are not proof of live use or deployment.
- [cmd/web.go](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/cmd/web.go): runWeb -> prepareWeb (settings, loggers, routes) -> graceful.InitManager -> serveInstalled -> routers.InitWebInstalled -> NormalHandler -> listen; observed source only, not tested.
- [README](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/README.md) reports prior test removal and limited smoke; this reset has not audited all test files, CI or production.
- [Previous settings ADR](https://github.com/datvtph41107/bdspro/blob/6fbb2104806d45ed1de8005609694acf0d1c7d1f/docs/architecture/decisions/ADR-G02-02-canonical-process-settings.md) is historical accepted scope; neither invalidated nor extended by reset.
- Old PR template links docs/engineering paths not present in inspected recursive dev tree; replaced by links to actual governance docs.
- Other historical docs retained unchanged or annotated at entry points. No claim that all docs or code have been audited.
- Business facts from owner: team capacity, incomplete requirements, absence of supplied traffic/incident evidence. Legacy six/seven services, gateway/RPC/protobuf reported by owner; exact old runtime topology and historical rationale not independently audited.
- Next: confirm a business workflow and acceptance examples. No architecture refactor or test result claimed.
