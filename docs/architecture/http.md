# BDSPro HTTP — explicit routes and controller inputs

## Why the DRX working pattern is useful

A developer can open `routes/auth.go` or `routes/api.go`, find an HTTP method and URL, follow the named controller method, examine the request type, and read a service/SQL query. This is the useful discipline from Laravel DRX — not PHP facades, hidden model binding, or a runtime that resolves controller methods by magic.

## HTTP responsibilities

| Place | Owner |
|---|---|
| `routes/*.go` | Native Gin route declarations: actual named controller methods and ordered middleware |
| `app/http/requests/` | Typed `LoginBody`, `VerifyEmailQuery`, `CreateOrganizationBody` |
| `app/http/request/` | `JSON(ctx,&body)`, `Query(ctx,&query)`, `PositiveID(ctx,...)` |
| `app/http/controllers/auth/` | Register, login, session, Phone OTP and Google HTTP actions |
| `app/http/controllers/api/` | Organization HTTP actions |
| `app/http/middleware/` | Authentication, source/identity rate limits, logging, request monitor, ingress |
| `app/http/resources/` | Public output fields only |
| `app/http/responses/` | Safe JSON errors and success writer |
| `app/http/server/` | Gin engine, CORS, socket, HTTP shutdown |

Controller code explicitly declares its input:

```go
var body authrequest.LoginBody
if !request.JSON(ctx, &body) { return }
if !c.limiter.Identity(ctx, "login", body.Email, 30, 10, time.Minute) { return }
result, err := auth.Login(/* ... */)
responses.Send(ctx, http.StatusOK, resources.NewLogin(result), err, "login failed")
```

The route uses direct methods, e.g. `router.POST("/users/login", middleware.SufLogin(), limiter.Source("login",30,time.Minute),authController.Login)`. There is no `routes.Handlers`, generic binding registry, `ctx.MustGet("request.login")`, auto controller resolver or `BaseController`.

Source rate-limiting happens before parse; the identity limiter runs explicitly **after** successful parse and before business processing. HTTP errors are written at the HTTP boundary, not inferred from arbitrary internal database messages. Database authority for Organization mutations remains enforced in the SQL predicate, not replaced by an HTTP permission guess.

Neither a pagination mechanism nor a filter builder is created without a real query consumer. For such a feature, name its request contract and add typed SQL parameters in `db/query/*.sql`; do not create a generic ORM or SQL string generator.

Legacy documents under `docs/engineering/` are historical artifacts, not instructions to reinstate a second input or route mechanism.
