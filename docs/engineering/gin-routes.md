# Gin route convention

Use Gin's **native** `GET`, `POST`, `PATCH` and `RouterGroup`. A group inherits its parent middleware; no middleware loop, custom `Route` facade, global router or reflection is required.

The only composition functions are in `internal/api/routes.go`: `registerRoutes`, `registerPublicRoutes`, `registerAuthenticatedRoutes`, `registerOrganizationRoutes` and `registerPhoneBindRoutes`. They receive an already configured Gin engine or group. A controller method is visible at the route line.

```go
s.registerOrganizationRoutes(route.Group("/organizations", middleware.VerifiedContact(s.store)))
```

```go
func (s *Server) registerOrganizationRoutes(route *gin.RouterGroup) {
    route.GET("", s.organizationHandler.Index)
    route.GET("/:id", s.organizationHandler.Show)
    route.POST("", s.organizationHandler.Store)
    route.PATCH("/:id", s.organizationHandler.Update)
}
```

Avoid local `authenticated := route.Group(...)`, `organizations := ...` and a per-file `sourceLimit := func(...)` alias. Avoid declaring authentication separately on every organization route: it can be missed accidentally. Keep the request chain ordered: global ingress → authentication → VerifiedContact → endpoint-specific middleware → controller → service → SQLC.

**Compatibility:** route/method set, optional Google/Phone availability, rate-limit operation names/limits, middleware ordering, success JSON and authentication remain unchanged. Registration runs only at startup; it creates no per-request wrapper beyond native Gin middleware.

Source rationale: Laravel DRX's `routes/api.php` is a discoverability reference, not a reason to copy PHP route facades into Go. `ginS` uses global engine state and is not a good fit for independently constructed test servers.
