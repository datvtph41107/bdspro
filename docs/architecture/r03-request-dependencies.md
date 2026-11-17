# R03 — APIContext initialization and request dependency lifetime

Reference:
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go
- https://github.com/go-gitea/gitea/blob/main/services/context/base.go

## BDSPro composition

1. `routers/api/v1.Routes` registers `AfterRouting(APIContexter())`.
2. `modules/web` compiles middleware and typed handlers at registration.
3. At request time, `modules/web/gin.go` attaches the matched Gin scope
   to `*http.Request`.
4. `services/context.APIContexter` establishes a request-local `reqctx`
   store when one does not already exist, then obtains exactly one
   `*APIContext` from the matched Gin context.
5. `GetAPIContext` publishes that instance to the standard HTTP request.
   `modules/web/handler.go` resolves it through the provider registered
   by `services/context.init()`; it does not construct another context.
6. Later middleware may attach enriched context values to the current
   HTTP request. `Base.Value` sees those values while `Done/Err/Deadline`
   are bound to the stable ingress cancellation context.
7. The request store's creator invokes cleanup on middleware exit,
   including when downstream code panics; nested APIContexter does not
   create a second store.

Unlike Gitea's HTTP-native `NewBaseContext(w, req)`, BDSPro currently
constructs `Base` through a single Gin RequestSource adapter. This is
intentional while Gin remains the matcher. BDSPro reuses a context already
created on the same Gin request; Gitea's Base constructor rejects a
duplicate Base construction. Both protect request-local ownership,
though their duplicate-handling policies differ.

## Invariants

- No business service constructs `APIContext` or calls `APIContexter`.
- No request-scoped object is stored in process global state.
- `modules/reqctx` owns request data cleanup.
- `services/context` owns the Base/HTTP response contract.
- `modules/web` owns the typed adapter and route composition.
- Request values may be enriched synchronously, but request cancellation
  must never depend on a mutable Gin request pointer.
- A middleware passing context to the next action must forward the
  enriched `*http.Request`.
