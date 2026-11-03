# R10 — Google OAuth HTTP code owned by focused Gin files

Gitea reference: `routers/api/v1/user` resource-specific handlers and service/model separation at commit `12b20c7`.

No replacement of Gin, no new OAuth mechanism, no change to JSON contracts, state cookie, nonce, PKCE verifier, provider token exchange, Redis state or error mapping.

- `google.go` — Google handler dependencies, shared request state and constructor.
- `google_start.go` — Google login/link/bind initiation, cryptographic state, verifier, cookies and Redis staging.
- `google_callback.go` — callback validation, provider exchange, linked-account session and safe errors.

All existing handler and helper function bodies are transferred unchanged. Go 1.27.3, OpenAPI, 34-route inventory, OAuth E2E tests, Redis, Docker and HTTP smoke remain the acceptance gate.
