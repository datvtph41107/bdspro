# Routers — request boundaries, not the whole web application

Gitea currently has `routers/init.go`, `routers/common`,
`routers/api`, `routers/web`, `routers/private`,
`routers/install` and `routers/utils`. These are different
HTTP surfaces and common helpers, not folders every backend needs.

- `routers/init.go`: installation/runtime initialization and composition.
- `routers/common`: HTTP protocol behavior reusable between transports.
- `routers/api`: external machine-oriented API, validation and contract.
- `routers/web`: independently rendered Gitea website.
- `routers/private`: internal authenticated endpoints for Git/SSH plumbing.
- `routers/install`: first-time web setup flows.

BDSPro has **one external REST transport** and two tiny HTML forms.
Keep `routers/init.go`, `routers/common` and `routers/api` only.
Static templates belong to `templates/`; they do not justify
`routers/web` or an extra router framework.

```text
Gin router (routers/api/server.go)
  -> routers/common.RequestID
  -> access log + Recovery
  -> API RequestMonitor/RequestIngress
  -> routers/api/routes.go
     -> security middleware Authenticate / rate-limit
     -> routers/api/handler
         parse -> services -> response
```

Where Gitea groups controllers by web/API and domain, BDSPro's current
`routers/api/handler` is deliberately small. Avoid a second route
registration package, reflective registry or unnecessary per-route wrapper.

All new routes must be discoverable in one route declaration file and
must have a method/path test and corresponding
`templates/swagger/openapi.json` operation.

Sources: [Gitea routers/init.go](https://github.com/go-gitea/gitea/blob/main/routers/init.go),
[routers/api](https://github.com/go-gitea/gitea/tree/main/routers/api),
[routers/common](https://github.com/go-gitea/gitea/tree/main/routers/common).
