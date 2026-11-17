# R02 — Typed handler contract

Gitea source:
- https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go

BDSPro target organization:
- `modules/web/router.go` owns `Router`, route scopes, `Methods` and `Mount`.
- `modules/web/handler.go` owns `toHandlerProvider`, static signature checks,
  typed argument resolution, middleware compilation and continuation rules.
- `modules/web/gin.go` is the single transport adapter while Gin remains in use.
- `services/context/api.go` registers and resolves request-scoped `*APIContext`.
- `services/context/base.go` owns request/response behavior; it is not a DI container.

Registration: `Router.Methods → wrapMiddlewareAndHandler →
toHandlerProvider → preCheckHandler`. Invalid signatures panic before serving.
Providers are only invoked at request time, after `APIContexter` has attached
the current `APIContext` to the HTTP request.

Request: `APIContexter → prepareHandleArgsIn → reflect.Call`. After a typed
middleware returns, the HTTP chain continues unless a response is committed
or the request context has been aborted. The handler adapter must reject
both a nil interface and a typed-nil provider result.

Gitea uses Chi and pre-checks typed handlers through a dry run with nil
request. BDSPro uses a registration-time signature check instead, because
it requires a Gin bridge and must not resolve live request dependencies
during registration. This is a deliberate implementation difference,
not a change in the user-facing handler syntax.

No global dependency resolver or new service initialization registry is
introduced by R02.
