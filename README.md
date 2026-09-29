# BDSPro

Clean-slate Go backend project.

## Status

This repository is the new working authority for BDSPro development and Go
learning from first principles.

The previous repository `datvtph41107/bdspro-backend` is preserved as historical
evidence only. Its architecture assumptions, package layout, service topology and
mentor-derived decisions are not automatically authoritative here.

## Current principle

Build from observed problems and Go/runtime behavior:

```text
problem
  -> mechanism
  -> smallest experiment
  -> real output
  -> explanation
  -> decision
  -> proof
```

Do not create architecture folders, packages, frameworks or infrastructure before
the problem that justifies them is understood.

## Repository identity

Current source host:

`github.com/datvtph41107/bdspro`

This is currently a personal GitHub namespace. The canonical Go module path is
intentionally **not initialized yet**. Module identity will be chosen only after
stable long-term ownership is decided (for example a real GitHub organization or
a domain that is actually controlled).

## Local workspace

Preferred local path:

`~/projects/bdspro`

## Durable continuity

Start from:

1. `docs/continuity/MASTER.md`
2. `docs/continuity/MEMORY.md`
3. `docs/continuity/CHECKPOINT.md`
4. `docs/continuity/DECISIONS.md`
5. `docs/continuity/HISTORY.md`

Research references live under `docs/research/`.
