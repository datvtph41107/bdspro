# R01.7 — APIContext without Gin

Gitea's `services/context/base.go` owns HTTP response/request semantics, `api.go` owns typed API state, and the router adapter handles HTTP framework integration.

BDSPro now follows the same ownership:
- `base.go`: transport-independent Base, request context, JSON/status, bind and response capability.
- `api.go`: APIContext and verified actor; no Gin import or Gin field.
- `writer.go` and `validation.go`: one error envelope, one public error classifier, one field validation path; no Gin import.
- `gin.go`: single Gin request adapter, APIContexter and Wrap.
- `gin_response.go`, `gin_validation.go`: legacy Gin Auth/OAuth helpers delegate to the SAME API writer/classification; no second response system.
- `modules/web/router.go`: typed routes; Gin remains sole middleware executor.

Request path: Gin request → APIContexter/Base → Authenticate → RequireSession → VerifiedActor → orgAssignment → route guards/bind → org HTTP action → service → model/SQLC → APIContext.JSON/APIError.

Constrained differences: Gin remains server routing middleware engine, SQLC remains database layer. Remaining legacy Organization/Team receiver methods, Auth/OAuth handlers are deliberately preserved for later migration. No global store, new transaction nesting or second writer introduced.

Gitea references:
https://github.com/go-gitea/gitea/blob/main/services/context/base.go
https://github.com/go-gitea/gitea/blob/main/services/context/api.go
https://github.com/go-gitea/gitea/blob/main/modules/web/router.go

Acceptance: Go 1.27.3 CI, API and middleware integration, error/validation parity, route audit, SQLC and Docker/HTTP smoke test.
