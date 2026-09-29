# Decision Register

Status vocabulary:

- OPEN
- HYPOTHESIS
- PROVISIONAL
- CLOSED
- SUPERSEDED

| ID | Question / decision | Status | Evidence | Next proof |
| --- | --- | --- | --- | --- |
| BDS-001 | Continue active development in a new clean repository `datvtph41107/bdspro` | CLOSED | Old workspace contains large historical topology, go.work context and residual source that interfere with clean Go learning/reasoning | Align local workspace and develop only in new repo |
| BDS-002 | Treat `datvtph41107/bdspro-backend` and its Rebuild V2 branch as historical evidence, not current architecture authority | CLOSED | User explicitly rejected inheriting earlier mentor assumptions without renewed analysis | Consult old source only when a concrete comparison/problem requires it |
| BDS-003 | Keep continuity, checkpoint and research in Git rather than conversational memory | CLOSED | Context/chat loss is expected and decisions must be reproducible | Recover future sessions from docs + live Git |
| BDS-005 | Do not create `go.mod` until BDS-004 is resolved | PROVISIONAL | Module path becomes import identity and is cheapest to choose correctly before code exists | Resolve BDS-004, then initialize module |
| BDS-006 | Use official Go sources and mature OSS as evidence rather than copying a repository tree | CLOSED | Moby, Terraform, Prometheus, Kubernetes and CockroachDB use different topologies for different pressures | Continue comparing mechanisms/problem context |
| BDS-007 | User types/runs implementation steps personally; explanations include purpose, mechanism and engineering consequence | CLOSED | Explicit learning goal is working reflex rather than pattern memorization | Enforce continuously |
| BDS-008 | Start the single executable as root `main.go`; do not create `cmd/` until command topology creates value | CLOSED | Official Go layout permits root package main; Terraform demonstrates root executable source at mature scale; Prometheus/Moby show `cmd/` when command surfaces grow | Reconsider only when a second command or mixed command/library pressure appears |
| BDS-009 | Do not create empty architecture directories (`internal/`, `pkg/`, domain/service/repository/etc.) | CLOSED | Official Go guidance and OSS comparison show these directories correspond to visibility, API, command or scale pressures rather than bootstrap requirements | Create the first directory only when code pressure identifies a responsibility |
| BDS-010 | Add repository-wide LF normalization through `.gitattributes` for the Windows/WSL workflow | CLOSED | Local proof showed Git resolved `text=auto` and `eol=lf`; a CRLF test file contained bytes `0d 0a` in the working tree and the staged Git blob contained only `0a`; the rule was committed and pushed | Keep the rule minimal unless a new repository-wide Git attribute problem appears |
| BDS-011 | Shared `.editorconfig` is part of the initial baseline | SUPERSEDED | Cross-repo evidence is inconsistent; Moby explicitly treats it as local customization while CockroachDB uses one. Go source formatting already has gofmt | Add later only if shared editor behavior solves an observed cross-editor/non-Go problem |
| BDS-012 | Makefile, Dockerfile and CI belong in initial bootstrap | CLOSED — REJECTED | Mature repositories have them because repeatable build/deploy/proof workflows exist; no such workflow exists yet in the clean repo | Introduce each only with its first concrete recurring contract |
| BDS-013 | Canonical repository naming model is `bdspro/bdspro`: owner namespace `bdspro`, flagship software-system repository `bdspro` | CLOSED | The name preserves the durable BDSPro identity without encoding legacy `bdspro-service`, backend/runtime topology, language or version; mirrors the owner/project separation seen in mature repositories while retaining distinct semantic layers | Operationalize only after the `bdspro` owner namespace is actually controlled |
| BDS-014 | Initial Go topology is one repository with one root module | CLOSED | The clean repository is currently one software/dependency/version unit; no capability has an independent release or consumer contract | Re-open only when an independently versioned/distributed module boundary is proven |
| BDS-004 | Canonical Go module path | OPEN | Repository/system naming is closed as `bdspro/bdspro`, but the exact import path cannot be finalized until the `bdspro` namespace is actually controlled | Once ownership is operational, initialize the root module with the controlled canonical path |

Do not erase old decisions. Mark superseded decisions and add the replacement.
