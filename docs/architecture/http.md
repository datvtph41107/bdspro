# HTTP ingress and API ownership

`routers/init.go` composes delivery-specific dependencies.
`routers/api/server.go` creates one native Gin engine;
`routers/api/v1/api.go` lists all methods and routes explicitly.
There is no custom route registry, ORM endpoint generator or second router.

`routers/common/` owns request ID and CORS behavior.
`routers/api/middleware/` owns authenticated admission, rate limits,
request limits and Redis-backed throttling.
`routers/api/v1/user/` and `routers/api/v1/org/` own Gin HTTP handlers. `modules/structs/` defines public request/response types; `services/convert/` maps persisted data to them.
`routers/api/response/` maps application error identities to API statuses.

`services/auth` and `models/organization` must not import Gin.
`models/db` handles persistence, not HTTP codes. The API prefix is
intentionally empty to preserve existing clients; changing it is a
public API migration, not a directory refactor.

## Source contract

- `routers/api/v1/api.go`: source of method/route registration.
- `templates/swagger/openapi.json`: reviewed OpenAPI 3.0.3 source.
- `routers/api/swagger_contract_test.go`: cross-checks both sources.
- `routers/api/swagger.go`: serves UI assets in development only.
- `templates/auth`: HTML for verification/reset (not a web frontend).

Future `routers/web` or `routers/private` requires a real second
transport and explicit authentication/security boundary.
