# Verified BDSPro feature map

The verified scope is intentionally small. Every feature has a route, runtime/persistence owner and an executable acceptance test. This is not a catalog of everything the application supports.

- [First-party sessions](sessions.md)
- [Organizations and team invitation](organizations.md)

To run the complete disposable lane from repository root:

```sh
bash .cursor/skills/verify-bdspro/scripts/verify.sh
```

Do not use a production or existing developer database. The runner owns its PostgreSQL/Redis containers, and the existing test helpers refuse non-disposable databases. `httptest` exercises actual Gin routing but does not prove TCP ingress or production deployment.
