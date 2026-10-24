# Register → Asynq → SMTP → mailbox → Verify (development)

The mail channel is intentionally exercised with real SMTP while the recipient
is captured by **Mailpit**, not forwarded to Gmail or the public Internet.

## Channels and ownership

| Channel | Owner | What to observe |
| --- | --- | --- |
| HTTP | Gin `POST /users` | HTTP 201, `X-Request-ID`, `account_id` |
| PostgreSQL | `RegisterTx`, `GetOrCreateVerifyEmailTx` | Account and verification rows |
| Redis | Asynq producer / scheduler | `scheduled`, `pending`, `active`, `retry` |
| Go worker | Asynq processor | `task_id`, `account_id`, `verification_email.completed` |
| SMTP | Mailpit listening on `mailpit:1025` | Accepted message shown at http://127.0.0.1:8025 |
| HTTP Verify | `GET /users/verify_email` | HTTP 200 and `accounts.is_email_verified=true` |

## Docker first-run and existing Redis safety

The current Compose has Redis AOF and a named volume. **If an older Redis
container still has queued tasks without a mounted volume, do not recreate it
blindly.** Inspect and migrate old Redis state first; see
`docs/operations/register-asynq.md`.

For an existing installation, sync source, inspect Redis, and start only Mailpit:

```bash
git fetch origin
git pull --ff-only origin dev

docker compose exec -T redis redis-cli INFO persistence
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:scheduled'
docker compose exec -T redis redis-cli LLEN 'asynq:{critical}:pending'
docker compose exec -T redis redis-cli LLEN 'asynq:{critical}:active'
docker compose exec -T redis redis-cli ZCARD 'asynq:{critical}:retry'

docker compose up -d --no-deps mailpit
docker compose up -d --no-deps --build api
docker compose ps
```

The API is now configured to send via `SMTP_ADDRESS=mailpit:1025`,
`SMTP_AUTH=false`, and `EMAIL_SENDER_ADDRESS=no-reply@bdspro.test`. No
Gmail credentials are required. The Mailpit UI is bound to the host loopback
interface; SMTP port 1025 is available **inside the Docker network only**.

## Make one real registration

Use a unique test email. Do not use a real user account or real password.

```bash
EMAIL="register-$(date +%s)@example.test"
echo "Test recipient: $EMAIL"

curl -sS -i -X POST http://127.0.0.1:8080/users \
  -H 'Content-Type: application/json' \
  --data "{\"email\":\"$EMAIL\",\"password\":\"example-password-123\"}"

docker compose logs --since=3m api | grep -E \
  'task enqueued|verification_email.completed|task failed|http.request'
```

HTTP 201 means registration committed, **not** SMTP delivered. Asynq uses
`ProcessIn(10s)`, so email appears later. Open http://127.0.0.1:8025 and
find the recipient address. Click the email verification link. It requests
`GET /users/verify_email` and should respond with `{"verified":true}`.

Inspect PostgreSQL (replace test address and account ID as needed):

```bash
docker compose exec -T postgres psql -U root -d bdspro -c \
  "SELECT id,email,is_email_verified,created_at FROM accounts ORDER BY id DESC LIMIT 5;"

docker compose exec -T postgres psql -U root -d bdspro -c \
  "SELECT id,account_id,is_used,created_at,expired_at
   FROM verify_emails ORDER BY id DESC LIMIT 5;"
```

Never print or paste `verify_emails.secret_code`, SMTP credentials, full
verification URLs, or raw message HTML into logs or tickets.

Mailpit API (read-only list of message headers and summaries):

```bash
curl -fsS 'http://127.0.0.1:8025/api/v1/messages?limit=10'
```

This API may expose recipient addresses and message snippets. Review before
sharing output.

## Gmail / actual outside mailbox

Mailpit does not deliver mail externally. To test real Gmail SMTP, run the Go
binary on your own trusted machine with `SMTP_ADDRESS=smtp.gmail.com:587`,
`SMTP_AUTH=true`, and valid `EMAIL_SENDER_ADDRESS` /
`EMAIL_SENDER_PASSWORD` process environment variables. Do not commit those
values, put them into a GitHub issue, or paste them into chat. Do not start
another API while Compose API occupies port 8080.

With Gmail, a successful SMTP send only proves **SMTP acceptance**. Final
inbox delivery, spam filtering, bounces, and provider-side restrictions must be
checked in the actual recipient mailbox/provider.

## Automated proof

CI runs `TestRegisterEmailEndToEnd` against disposable PostgreSQL, Redis and
Mailpit containers. It verifies Register HTTP 201, PostgreSQL account creation,
SMTP receipt at Mailpit, the actual emailed verification link, and the final
`is_email_verified=true` database state.

Limitations: the PostgreSQL-Redis handoff is not atomic, SMTP can accept mail
before a transport error is returned, and no provider delivery/bounce tracking
is implemented.
