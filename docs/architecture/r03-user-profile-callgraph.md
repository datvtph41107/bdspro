# R03 — Gitea-style user/profile actions and request-scoped dependencies

BDSPro base: `dev@ed3a78e98c89a78fbe91668c2fd923a73866c27a`.
Gitea reference: `0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a`.

## Gitea call sites and ownership

- `routers/api/v1/user/user.go.GetAuthenticatedUser(*context.APIContext)`
  converts the authenticated `ctx.Doer` to a public response. It does not
  instantiate an HTTP handler object.
- `routers/api/v1/user/settings.go.GetUserSettings` renders a projection.
- `routers/api/v1/user/settings.go.UpdateUserSettings` retrieves
  `web.GetForm[*api.UserSettingsOptions]`, constructs `services/user.UpdateOptions`,
  calls `services/user.UpdateUser`, then `ctx.JSON` or
  `ctx.APIErrorInternal`. This updates a preloaded XORM user model.

Sources:
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/user/user.go
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/routers/api/v1/user/settings.go
- https://github.com/go-gitea/gitea/blob/0ab0bfc375cc2b33b2a878b27ca5ee60f28f774a/services/context/api.go

## BDSPro equivalent

HTTP route admission:
`routers/init.go.NormalRoutes`
 -> `services/context.APIContexter`
 -> `routers/api/v1.registerAuthenticatedRoutes`
 -> `services/context.Authenticate`
 -> `services/context.RequireSession`
 -> typed `routers/api/v1/user.GetAuthenticatedUser/UpdateProfile/GetUserSettings/UpdateUserSettings`.

Binding:
`modules/web.BindJSON[T]` -> `reqctx` per-request form store ->
`web.GetForm[*api.ProfileUpdateRequest/SettingsUpdateRequest]`.

Business:
`services/profile.actingProfile` obtains the `models/auth.Actor` from
the live request context and the injected `db.Store` from `models/db`.
`UpdateCurrentProfile/UpdateCurrentSettings` reuse the original
`Service.Update/UpdateSettings` transaction implementation; those methods
call `models/auth.RequireActingSession` in the same SQLC transaction as
profile/settings updates. A PostgreSQL commit failure publishes no update.

Responses:
`services/convert.ToAccountProfile/ToAccountSettings` ->
`services/context.Base.JSON` -> the single Gin codec. Errors pass through
`APIErrorAuto -> PublicError -> modules/util.ErrorForClient` and the
typed `APIErrorResponse`; internal causes remain private.

## Decisions and caveats

- Remove `ProfileHandler` and its constructor: no handler-specific Store.
- Maintain Go Gin/SQLC rather than copy Gitea's Chi/XORM implementation.
- Do not trust signed claims without `RequireSession` for first-party
  routes. Commit-order authorization stays inside the profile service.
- Use the common route-level `bind[T]` and `ctx.Validation` mechanism.
  This standardizes malformed JSON/field-validation responses at the API
  boundary and can choose a more specific public error code than the old
  `ctx.APIInvalid` handler; keep HTTP 400 and verify the change explicitly.
- Test replays a request admitted by session middleware but revoked before
  the update's transaction, using the same fake SQLC transaction as before.
  It must fail without writing profile or settings.

## Next steps

Auth/OAuth remains the larger legacy Gin receiver surface:
`routers/api/v1/user/auth.go`, `oauth2.go`,
`routers/api/v1/oauth`, `routers/web/oauth`.
Their cookie, redirect, CSRF, rate-limiting and OAuth2 response semantics
require separate route-by-route traces before converting action signatures.
Also audit `routers/init.go` versus Gitea's
`InitWebInstalled/NormalRoutes/mustInit` and avoid starting web
dependencies from worker/migrate commands.
