# C03.1 — Typed HTTP grouping, request form ownership and Team listing

## Grounded reference

- Gitea `modules/web/router.go` owns route grouping, middleware compilation and binding within a Chi router. `routers/api/v1/org/team.go` contains plain functions, such as `ListTeams(ctx *context.APIContext)`, using already-assigned Organization state.
- BDSPro `dev@b0d3b53800a59961b1ff2248a23d739e3b228053` already has a minimal Gin-backed `modules/web.Router[T]` and typed functions for Team creation/read, but `GET /orgs/:id/teams` still called the `TeamHandler.List` receiver. Form binding was split manually between `modules/web/middleware.BindJSON`, `web.SetForm` and an API-boundary `bind` function.
- Source: https://github.com/go-gitea/gitea/blob/main/modules/web/router.go
- Team reference: https://github.com/go-gitea/gitea/blob/main/routers/api/v1/org/team.go

## Decision

1. Keep Gin as the **only middleware executor** and preserve the existing `Router[T]` typed action adapter. Chi's `BeforeRouting`/`AfterRouting` semantics are not identical to Gin's `Use` and route-group chains. Do not implement an inaccurate second routing layer to mimic method names.
2. `modules/web.BindJSON[T]` combines typed JSON binding and request-scoped form storage, publishing only successful bindings. The API-boundary helper still owns public validation error rendering. No `services/context` import may enter `modules/web`.
3. Convert one production endpoint `GET /orgs/:id/teams` from `TeamHandler.List` to `org.ListTeams`. Its Group loads the assigned Organization through `orgAssignment(nil)`; the action uses `VerifiedActor` and `db.StoreFromContext` to call existing account-filtered SQLC-backed model operations. No second authorization algorithm, transaction API, converter or output schema.
4. Move the pagination helper into `org/team.go`, shared with the remaining legacy member route. Remove the unused `TeamHandler.List` method; leave other receiver actions until their own parity acceptance.
5. An absent context accessor in `NewRouter` is a wiring error and must fail during registration rather than at a later HTTP request.

## Acceptance

- Typed bind stores the exact decoded pointer on valid JSON and never exposes a form after invalid JSON.
- Malformed pagination returns 400; malformed organization path is rejected by assignment before any DB access.
- Existing `TestTeamInvitationAuthorizationAndReplayHTTP` covers a visible empty team page, outsider 404 and a writable team by owner through real Gin/SQLC/PostgreSQL.
- Existing route inventories and OpenAPI checks must remain identical.
- Checks must run on the actual PR head; a green CI result is only evidence for the tests run, not proof that every Organization action has migrated.

## Limits and next slices

- Remaining `TeamHandler` and `Handler` methods remain because their consumers are still registered; deleting them now would be cosmetic and break behavior.
- `APIContext.Doer` retains signed claims; `VerifiedActor` is authority from the PostgreSQL-backed session. Do not adopt Gitea's `Doer *User` blindly.
- Organization preflight is visibility/early denial, not commit-order write authority; SQLC transaction predicates remain mandatory.
- `services/organization.TeamService` and direct model entrypoints need consumer mapping and parity tests before being replaced with context-scoped operations.
