# R08 — Response ownership and public error boundary

Reference:
- https://github.com/go-gitea/gitea/blob/main/services/context/base.go
- https://github.com/go-gitea/gitea/blob/main/services/context/api.go
- https://github.com/datvtph41107/bdspro/blob/dev/services/context/base.go

## The response owner

`services/context.Base` is the only application response contract.
`Base.JSON` encodes with `modules/json.MarshalWriteRelaxed` into a
buffer before the Gin adapter emits the bytes. A serialization error
returns a fixed 500 response; it must not expose serializer details.

The transport's `WrittenStatus` is zero until HTTP headers are committed.
Calling `Status(202)` alone may select a status without committing it;
a subsequent `JSON(200, payload)` may still produce HTTP 200.

Once committed, `JSON` and `Status` must not write or replace another
response. A late `APIError` aborts the remaining chain, but cannot
replace an already delivered response and must not mutate its error code
metadata. This is a defense-in-depth invariant, not permission for
handlers to continue after writing a response.

## Public error classification

`services/context.APIErrorAuto` uses `PublicError`, which recognizes
only `modules/util.WithPublic` errors and maps their category to an
HTTP status (400/401/403/404/409/429). Unknown errors are logged with
type metadata and sent as `500 {"error":"internal server error",
"code":"internal_error"}`. Internal driver/provider messages are not
included in the public response.

`ctx.Validation` owns parsing/validation response details and follows
the existing API 400 contract; this remains distinct from a domain
error returned by a service.

## Runtime tests

`TestR08JSONCommitIsSingleResponse`: second JSON and late status
selection cannot overwrite or append to a committed success.

`TestR08ErrorAfterCommittedResponseAbortsWithoutOverwriting`: a late
error does not append a second JSON object and aborts future actions.

`TestR08JSONEncodingErrorReturnsSafe500AndStopsChain`: failure to
serialize a value produces a safe 500 and stops subsequent typed actions.

No changes are made to domain service signatures or the application's
success response envelope.
