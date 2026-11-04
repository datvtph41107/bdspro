# JSON v2 — accepted BDSPro serialization boundary (R30)

Sources: [Gitea json.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/json/json.go),
[Gitea jsonv2.go](https://github.com/go-gitea/gitea/blob/ab295e62e4c7/modules/json/jsonv2.go).
See [R30 source and ownership review](r30-auth-json-timeutil.md).

## What remains and why

`modules/json/json.go` exposes precisely `Marshal`, `Unmarshal` and
`NewDecoder` (plus the decoder's `Decode` method). Those APIs are called
by cache JSON helpers, persisted Asynq task writers/readers, and the OAuth2
state/token/JWKS data path.

`modules/json/jsonv2.go` owns Go 1.27 JSON v2 implementation details:
`FormatNilSliceAsNull` and `FormatNilMapAsNull` maintain prior persisted
wire formats; `DefaultOptionsV2` uses strict duplicate-name, UTF-8 and
case-sensitive property matching.

Gitea also exposes deterministic encoding, legacy optional-empty behavior,
relaxed UTF-8, JSON pretty printing, and a configurable default JSON
interface. There is **no corresponding production caller in BDSPro**.
Do not copy them or introduce a mutable global codec registry in anticipation
of hypothetical features.

## Compatibility and security contract

- Existing correctly encoded JSON v1 OAuth2 state and Asynq payloads must
  remain decodable; exact old/new bytes are tested where data is persisted.
- Duplicate object properties and malformed UTF-8 are rejected. This
  intentionally tightens parsing of ambiguous inputs.
- Gin continues owning HTTP response encoding; JWT claim validation and
  signature verification remain in their respective libraries.
- `json.NewDecoder` is for stream decoding provider HTTP data already
  constrained by `io.LimitReader`. It does not itself enforce a size limit.
- Do not treat JSON decoding as domain validation: claims, IDs, expiry,
  authorization and transaction rules are checked by their existing owners.
- Add a public codec method only after identifying its consumer, operational
  semantics and regression test.

Test coverage: `modules/json/json_test.go`,
`services/auth/json_contract_test.go`, and
`services/worker/json_contract_test.go`.
