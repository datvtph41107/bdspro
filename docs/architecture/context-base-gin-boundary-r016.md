# R01.6 — Base HTTP contract independent of Gin

Source-level intent from Gitea:

- `services/context/base.go` owns the base request contract, not Gin or application business dependencies.
- `services/context/base_path.go` owns route/path accessors.
- `services/context/api.go` owns `APIContext` lifecycle and public API-specific fields.
- `services/context/gin.go` is **the only bridge** adapting a Gin request to `Base.RequestSource`.
- `models/auth.Actor` in request context remains the authority; `Doer` signed JWT payload is retained as a compatibility view. `VerifiedActor` requires a matching live session.
- `modules/web` composes typed actions at registration; Gin remains the only middleware executor.
- `services/context/writer.go` remains the single public response/error writer.

Unlike Gitea's Chi/reqctx-based `Base`, BDSPro uses a narrow `RequestSource` contract because Gin middleware can replace `*http.Request` when installing session and transaction scope. The source always yields the current request, so Base cannot retain a stale context value. No global State or mutable Store is introduced.

The `*gin.Context` field is intentionally retained in **APIContext only** while existing OAuth, account and membership actions still use Gin helpers. Removing it requires migrating all consumers and tests; copying a new response writer in parallel would violate the one-source-of-truth requirement.

Gitea evidence:
- https://github.com/go-gitea/gitea/blob/main/services/context/base.go
- https://github.com/go-gitea/gitea/blob/main/services/context/base_path.go
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go
- https://github.com/go-gitea/gitea/blob/main/modules/web/handler.go

Acceptance: APIContext must still implement `context.Context`, HTTP actions preserve the old Gin API during migration, Base itself must not import Gin, typed actor/path tests pass, and CI route, Docker, DB and HTTP tests remain green.
