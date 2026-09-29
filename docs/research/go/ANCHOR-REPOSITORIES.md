# Go/Open-Source Anchor Repositories

Updated: 2026-09-29

These repositories are temporary engineering anchors for BDSPro. They are not
templates and do not define BDSPro architecture.

For every comparison, ask:

```text
what problem does this repository have?
what pressure caused this structure?
what mechanism solves it?
does BDSPro have the same pressure now?
what would copying it cost?
what is the smallest experiment that can prove the idea here?
```

## Tier A — Daily anchors

### 1. Moby

Repository: https://github.com/moby/moby

Study:

- repository/module identity;
- root executable codebase vs independently versioned public modules;
- client/API boundaries;
- `internal/` and `pkg/` in a mature system;
- migration cost when public module identity changes.

Use when BDSPro asks:
"Is this a module/release/API boundary or merely a source folder?"

### 2. Terraform

Repository: https://github.com/hashicorp/terraform

Study:

- mature root `package main` usage;
- extensive `internal/` implementation boundaries;
- CLI composition;
- dependency ownership;
- large application structure without treating `cmd/` as mandatory.

Use when BDSPro asks:
"Can the executable stay at root, and what supporting code should become private
packages?"

### 3. Prometheus

Repository: https://github.com/prometheus/prometheus

Study:

- capability-oriented packages such as scrape/rules/storage/discovery;
- multiple command entry points;
- application repository vs reusable library repositories;
- build/test/release workflow growth.

Use when BDSPro asks:
"Has the application grown enough to justify command separation or a new
capability package?"

### 4. etcd

Repository: https://github.com/etcd-io/etcd

Study:

- distributed state;
- Raft/consensus integration;
- storage and server boundaries;
- client/server/API separation;
- multi-module/workspace evolution.

Use when BDSPro asks:
"What changes when state, coordination, failure recovery or distribution becomes
the actual problem?"

### 5. NATS Server

Repository: https://github.com/nats-io/nats-server

Study:

- a production server with root `main.go`;
- networking and concurrency;
- lifecycle/shutdown;
- performance-sensitive server code;
- configuration and operational concerns.

Use when BDSPro asks:
"What does a focused long-running Go server look like without assuming many
microservices?"

## Tier B — Problem-specific anchors

### 6. Temporal

Repository: https://github.com/temporalio/temporal

Study:

- durable execution;
- retries and failure semantics;
- service composition;
- persistence abstractions;
- background processing and distributed workflows.

Use when BDSPro reaches:
jobs, workflows, retries, idempotency, asynchronous failure recovery.

### 7. Caddy

Repository: https://github.com/caddyserver/caddy

Study:

- extensible module/plugin architecture;
- configuration as an explicit model;
- command vs reusable core;
- long-running process lifecycle;
- how extensibility is earned without turning every feature into a service.

Use when BDSPro asks:
"Should this capability be dynamically extensible or statically composed?"

### 8. Traefik

Repository: https://github.com/traefik/traefik

Study:

- dynamic configuration;
- provider boundaries;
- HTTP routing/proxy concerns;
- runtime reconciliation;
- integration-heavy infrastructure software.

Use when BDSPro reaches:
gateway/routing/config-watch/provider integration problems.

### 9. Kubernetes

Repository: https://github.com/kubernetes/kubernetes

Study only when scale justifies it:

- canonical module identity distinct from Git hosting;
- staging/workspace/publication boundaries;
- API machinery;
- controllers/reconciliation;
- ownership/governance at massive scale.

Do not copy its root topology into BDSPro.

Use when BDSPro has a real:
controller/reconciliation/API versioning/large multi-module problem.

### 10. CockroachDB

Repository: https://github.com/cockroachdb/cockroach

Study only when scale justifies it:

- very large package graph;
- transactional/concurrency behavior;
- distributed SQL/storage architecture;
- custom build tooling;
- package naming and ownership under extreme scale.

Do not copy Bazel or its root surface merely because it is production-grade.

Use when BDSPro reaches:
high-contention transactions, distributed data, large build graph or developer
tooling pressure.

## Default comparison order

For ordinary BDSPro work:

```text
Go standard library / official docs
  -> Moby
  -> Terraform
  -> Prometheus
  -> etcd or NATS depending on problem
```

Only then pull in:

```text
Temporal / Caddy / Traefik / Kubernetes / CockroachDB
```

when the current problem actually matches them.

## Root-working rule

The root is not populated by copying the union of these repositories.

For every proposed root item X:

```text
1. What repository-wide problem exists now?
2. Who/what owns that problem?
3. What happens if X does not exist?
4. Is the problem recurring or high-risk enough to justify a root contract?
5. Which anchor repository has the same pressure?
6. What is different in BDSPro?
7. What is the smallest version of X that solves only the current problem?
8. How do we prove it works?
9. Commit that responsibility separately.
10. Re-open the decision only when new pressure appears.
```
