# A02 — Trusted proxy contract

The API records login source IPs and uses Gin's `ClientIP()` for authentication throttling and monitoring. Reverse-proxy deployment therefore changes **security semantics**, not just logging.

- `TRUSTED_PROXIES=` (default) means **do not trust forwarding headers**. Gin reads the direct TCP peer as the client.
- Behind a known ingress proxy, configure **only its immediate peer addresses/CIDRs**, separated by commas, e.g. `TRUSTED_PROXIES=10.10.0.5,10.10.1.0/24`. These are examples, not production defaults.
- Missing/malformed entries and catch-all `0.0.0.0/0` or `::/0` fail server initialization. Do not silently ignore invalid security configuration.
- Do not make the API port directly reachable from untrusted clients when proxy IPs are trusted. Otherwise an attacker can impersonate an ingress peer and spoof forwarded headers.
- Proxy configuration must be reviewed together with network ACLs, real reverse-proxy forwarding behavior and tests in the deployment environment. This patch does not register devices or prove source IP ownership.

## Acceptance

`go test ./internal/api -run 'Test(TrustedProxyClientIP|InvalidTrustedProxyConfigFailsStartup)'`

Tests cover directly connected clients, spoofed `X-Forwarded-For` headers from untrusted peers, configured trusted peers and fail-closed configuration errors. Full `go test ./...`, race, vet, build and CI must run against the actual head before merging.

This is an **A02 ingress sub-slice**. Persistent login success/failure history, device observation, retention, and session policy still require separate work; it does not close A02.
