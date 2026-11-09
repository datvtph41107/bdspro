# R01 — Source-traced router, context and error pattern

Base: BDSPro `dev@9f0f6e875a4ad32759d63167df1791a061ed3296`
Reference: Gitea `0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a`.

Primary source:
- [Gitea modules/web/handler.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/web/handler.go)
- [Gitea services/context/api.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/api.go)
- [Gitea services/context/base.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/base.go)
- [Gitea modules/json/jsonv2.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/json/jsonv2.go)
- [Gitea modules/util/error.go](https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/modules/util/error.go)

## Call graph and ownership

Gitea: `routers/api/v1/org.CreateTeam`
 -> `services/org.NewTeam`
 -> `models/organization`
 -> error return -> `services/context.APIErrorAuto`
 -> `modules/util.ErrorUnwrapForUser`
 -> `APIContext.APIError` / `APIErrorInternal`
 -> promoted `Base.JSON`
 -> `modules/json.MarshalWriteRelaxed`
 -> `JSONv2.MarshalWriteRelaxed`
 -> `services/context.Response.WriteHeader`
 -> `modules/web.hasResponseBeenWritten`
 -> handler chain stops.

BDSPro: `routers/api/v1/org.CreateTeam`
 -> `services/organization.NewTeam`
 -> session and owner authorization inside SQLC transaction
 -> `models/organization.InsertTeam`
 -> `services/context.APIErrorAuto`
 -> `PublicError` and `modules/util.ErrorForClient`
 -> typed `APIErrorResponse`
 -> `Base.writeError`
 -> `ginRequestSource.AbortJSON` -> Gin JSON codec.
Gitea's `message/url` is not BDSPro's `error/code` public contract.
`util.WithPublic` permits public detail only when explicit.
`util.WithCause` preserves private DB cause for `errors.Is/As`.
Do not change either to plain `err.Error()`.

## Verified behavior difference addressed

Gitea's typed dispatcher checks `WrittenStatus()!=0` before running
the next handler. BDSPro's Gin adapter previously called every typed
action unless Gin `Abort()` was explicit. Gin's ordinary `JSON` does not
abort subsequent actions. `modules/web.Router.handlers` now calls
`c.Abort()` after an action **commits** the response, not after selecting
an uncommitted status. The rule is implemented once at the typed registration
boundary, not copied into every endpoint.

`routers/api/v1.registerTeamRoutes` now groups all `/:team_id`
endpoints under one path group. Crucially, `teamAssignment` remains on
the GET action only; applying it to all team mutations would change access
semantics and add extra SQLC queries. Path and HTTP method are unchanged.

## Intentional differences from Gitea

Gin remains the one router and middleware executor: no Chi/reflective
dispatcher or second response writer. BDSPro `modules/json` owns
persisted OAuth, queue and cache formats; Gin owns HTTP output.
The code intentionally avoids Gitea's relaxed UTF-8 writer without a
production HTTP codec requirement and compatibility tests. Zerolog already
owns request diagnostics; no Gitea logger manager clone is introduced.

## Tests and gates

- A response emitted by a typed action stops subsequent actions.
- A committed group guard stops its endpoint.
- Setting a status without writing headers does **not** stop the endpoint.
- Public errors retain exact HTTP status and `error/code` envelope.
- Wrapped PostgreSQL causes and unexpected errors remain private.
- Code formatting, full route inventory and Go 1.27.3 `go test ./...`
  and `go test -race ./...` are acceptance gates, not inferred outcomes.

Follow-up: rename actual service ownership to `services/org` consistently,
trace remaining user/Auth/OAuth Gin actions into typed API actions, remove
unused legacy team handlers only after callers/tests are migrated, and
audit `routers/init.go` ingress plus `modules/web` binding/validation.
