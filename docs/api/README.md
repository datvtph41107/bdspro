# REST API and Swagger / OpenAPI

**Canonical external API contract:** `templates/swagger/openapi.json`,
a reviewed OpenAPI 3.0.3 document.

**Actual API implementation:** `routers/api/routes.go`, native Gin.
The contract test `routers/api/swagger_contract_test.go` checks that every
documented HTTP method/path corresponds to a registered runtime route,
including optional provider routes, and vice versa. It does not verify
every response field or every status code; those need dedicated tests.

In development, `routers/api/swagger.go` serves:
- `GET /swagger/`: embedded Swagger UI.
- `GET /swagger/openapi.json`: reviewed OpenAPI JSON.

No Swagger endpoint is registered in production mode.

`templates/swagger/` contains the reviewed spec and the UI distribution,
including its upstream license notices. `templates/swagger.go` embeds the
assets into the executable. It does not rely on CWD, a mounted Swagger folder,
or a separate Statik-generated Go package.

```sh
make swagger-check
make swagger
go test ./routers/api -run TestOpenAPICoverage -count=1
```

`make swagger` currently runs contract validation. The former
`swag init` generated Swagger 2.0 material that the running UI did **not**
consume, and `statik` created unused generated Go. Both were removed,
not secretly replaced with a new generator. If source-generated OpenAPI is
required, decide its canonical source/format and test byte-for-byte
reproducibility before restoring generator commands.

API changes must update the route declaration, handler/schema, OpenAPI and
tests in one PR. Do not create both `doc/swagger` and
`templates/swagger` as competing sources.
