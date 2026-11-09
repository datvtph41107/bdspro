# R02 — Canonical Team actions and Gitea source-level dependency trace

Base: BDSPro `dev@6542ebd40c90dab490ff65d338baaec139156922`.
Reference: Gitea `main@0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a`.

## Gitea: what the code actually calls

1. `routers/api/v1/api.go` registers `/teams/{teamid}` with assignment,
   `bind(api.EditTeamOption{})` and `org.EditTeam`.
2. `routers/api/v1/org/team.go.EditTeam(ctx *context.APIContext)` retrieves
   `web.GetForm[*api.EditTeamOption](ctx)`, reads `ctx.Org.Team`, then
   calls `services/org.UpdateTeam`, `services/convert.ToTeam` and
   `ctx.JSON`.
3. `services/org/team.go.UpdateTeam` owns the XORM transaction and
   cross-model repo/team access recalculation. Model `models/organization.Team`
   owns the entity. `services/context.APIErrorAuto` translates errors
   through `modules/util.ErrorUnwrapForUser`; `Base.JSON` writes
   through `modules/json.MarshalWriteRelaxed`.
4. `modules/web/handler.go` prevents invoking another typed handler
   after the context reports a committed HTTP response.

Source links:
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/api.go
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/org/team.go
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/org/team.go

## BDSPro: concrete equivalent, not XORM/Chi replacement

`routers/api/v1/api.go.registerTeamRoutes` -> typed form binder in
`modules/web` -> package function `org.EditTeam(ctx)`
-> `ctx.PositivePathID` -> `web.GetForm[*api.RenameTeamOption]`
-> `services/org.RenameTeam(ctx,teamID,name)` -> `actingOrganization`
-> existing `TeamService.RenameTeam` transactional implementation
-> `models/auth.RequireActingSession` -> `models/organization.RenameTeam`
-> SQLC `RenameTeamForOwner` -> `services/convert.ToTeam` -> `ctx.JSON`.

Invitation actions:
- `org.CreateInvite(signingKey)` captures an immutable startup secret.
- `org.AcceptInvite` uses a bound `AcceptTeamInviteOption`.
- `org.RevokeInvite` uses `PositivePathID` and UUID parsing.
- All three call `services/org` package functions that obtain the current
  actor and injected `db.Store` from the request context, then use the
  pre-existing `TeamService` transaction methods (single authority).
- The raw invitation proof never enters the public response; the outbox
  enqueue intent is committed with the invitation state.

## Decisions and constraints

- Removed `routers/api/v1/org/team_legacy.go` receiver handlers. Neither
  the router nor tests construct a stateful `TeamHandler` anymore.
- Keep `TeamService` compatibility methods because integration tests and
  transactional callers still exercise them; package functions are thin
  request-context adapters, **not alternative transaction implementations**.
- Do not add the secret to `APIContext`, a global setting variable or a
  singleton. The closure is created once at route registration.
- Preserve method/path pairs and external `error/code` response.
- Bind validation precedes typed action invocation, matching Gitea's
  route-level binding. Previously some legacy methods parsed path before
  binding. If both path and body are invalid, the selected 400 detail can
  differ; this precedence must be tested before treating as strict parity.
- Gitea's `ctx.Org.Team` can be preloaded; BDSPro SQLC mutation checks
  owner and acting session within the commit. Preloading every mutable
  Team solely to mimic Gitea would add a query and risk snapshot confusion.

## Verification

Test package actions against malformed bind input, team ID and invitation UUID
without DB access. In full CI assert Go 1.27.3 gofmt, sqlc, architectural
ownership, full tests, race detector, route inventory, Docker and smoke.
This slice changes neither schema nor Gin/SQLC transaction semantics.

## Next source-trace slice

`routers/api/v1/user` mixes standalone `profile` actions and Gin receiver
methods for Auth/OAuth. Trace `appctx.GinActions` to its middleware caller,
`modules/web.BindJSON`, `services/context.APIContexter`,
`models/auth`, `services/auth` and the public error/JSON pipeline before
removing the remaining compatibility adapter. Audit `routers/init.go`
bootstrap against Gitea `InitWebInstalled/NormalRoutes`, but preserve the
separate worker/migrate command lifecycles.
