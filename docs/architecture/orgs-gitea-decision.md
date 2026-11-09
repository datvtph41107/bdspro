# R01.3 — Organization route naming under Gitea-style ownership

Gitea uses short `/orgs` HTTP names with `org` actions and full `organization` model names. BDSPro follows the same semantic split, while retaining its Party–Person–Account domain constraints and Gin/SQLC runtime.

- `routers/api/v1/api.go` registers HTTP routes and security middleware; the canonical path is `/orgs`.
- `routers/api/v1/org` remains the owner of HTTP action code.
- `models/organization` and `services/organization` retain the full domain name.
- PostgreSQL `organizations`, `organization_members` and `teams` stay unchanged.
- API `organization_id` and domain error codes remain unchanged.
- No implicit `/organizations` alias is registered.
- Historical route snapshots are not rewritten: `routes-orgs-additions.json` and `routes-orgs-combined-removals.json` document the delta.

**Review gate:** full build on Go 1.27.3, OpenAPI route parity, live-session and owner-revocation concurrency tests, CI success and reviewed API compatibility impact before merging to `dev`.
