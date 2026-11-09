# HTTP context ownership (Gitea comparison) — R01.5

## One source of truth

The API request lifecycle has distinct authorities:

| Boundary | Owner | Contract |
|---|---|---|
| Signed JWT parse | `services/context/auth.go` | `Doer` is a JWT compatibility view, **not live session authority** |
| Session verification | `services/context/session.go` | `models/auth.Actor` is installed only after PostgreSQL verifies a live first-party session |
| HTTP → domain bridge | `APIContext.VerifiedActor()` in `services/context/api.go` | Check `IsSigned`, signed JWT identity, and the matching actor in request context |
| Resource assignment | `routers/api/v1/organization_context.go` | Use the verified actor for account-scoped SQLC reads |
| Path parsing | `services/context/base_path.go` | Positive int64 parsing once; caller determines HTTP/domain error |
| Authorization of writes | `services/organization`, `models/organization`, SQLC | Transaction-bound, account/owner/session locks remain authoritative |
| HTTP errors | `services/context/writer.go` | Existing public envelope, no raw SQL/credential leaks |

Do not confuse `Doer` with `models/account` or Gitea's `*user_model.User`: BDSPro's `Doer` is currently a signed payload for compatibility. Do not promote a JWT to an authorized actor without PostgreSQL session validation.

## Gitea correspondence

Gitea defines `Base` in `services/context/base.go`, URL accessors in `base_path.go`, and `APIContext` in `api.go`. Gitea's API authentication prepares `Doer` and `IsSigned`; resource assignments read from typed request context. BDSPro retains Gin and SQLC, and requires live-session identity for `VerifiedActor`.

References:
- https://github.com/go-gitea/gitea/blob/main/services/context/base.go
- https://github.com/go-gitea/gitea/blob/main/services/context/base_path.go
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go
- https://github.com/go-gitea/gitea/blob/main/routers/api/v1/api.go

## Explicit limits

- Legacy HTTP handlers still read JWT `Doer` after parent middleware verifies sessions.
- `Base` still embeds Gin to preserve existing consumers; its eventual decoupling is a separate review.
- The new guard does not replace in-transaction session/owner checks.
- No additional database query is made to compare actor and Doer: identity is compared against verified request-local state.
