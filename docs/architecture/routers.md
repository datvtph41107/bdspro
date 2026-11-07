# Routers — request boundaries, not the whole web application

Gitea separates its router surfaces, request context, services and technical
modules. BDSPro retains one Gin REST engine plus browser OAuth endpoints,
without adopting another routing framework.

- `routers/app.go` builds HTTP dependencies and the initialized Gin engine.
- `routers/init.go` installs the shared ingress chain and recovery.
- `routers/common` provides request ID and CORS.
- `routers/api/v1/api.go` declares endpoint method/path pairs once.
- `routers/api/v1/{user,org,oauth}` handles protocol input and representations.
- `routers/web/oauth` owns the real browser OAuth pages and cookie transport.
- `services/context` owns JWT principal, verified contact admission,
  APIContext, binding/error responses, and request body cap.
- `services/security` owns HTTP auth throttle, denied-request observation,
  and login outcome logging.
- `modules/ipactive` implements action-based rate-limit admission.

```text
Gin (routers.NormalRoutes)
 -> routers/common.RequestID
 -> access log + Gin recovery
 -> security.RequestMonitor -> context.RequestIngress
 -> routers/api/v1/api.go
    -> context.Authenticate / VerifiedContact (route-specific)
    -> IPActive and security.AuthThrottle (operation-specific)
    -> endpoint -> service or model -> response contract
```

`APIContext` wraps the existing Gin request context without another router or
middleware executor. `Doer` contains verified first-party access JWT claims,
not a fresh database account record; live state checks are explicit.

SQL owner/membership predicates, transaction locks, OAuth token handling,
and public method/path, error code and response contracts must not change
because of package reorganization. The directory name `v1` does not
automatically add a URL prefix.

New public routes need the route inventory and matching
`templates/swagger/openapi.json` operations. See [HTTP ownership](http.md).

References: [Gitea routers](https://github.com/go-gitea/gitea/tree/main/routers),
[Gitea context](https://github.com/go-gitea/gitea/tree/main/services/context).
