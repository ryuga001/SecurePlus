# SecurePlus

An outbound email security gateway. Mail from a tenant's mail system is accepted over SMTP, queued, evaluated against that tenant's data-loss-prevention policies, DKIM-signed, relayed to the recipient's MX, and recorded — with a dashboard over the resulting delivery audits and policy incidents.

## How mail flows

```
Exchange / Gmail
       │  SMTP
       ▼
┌───────────────┐  authorize sender domain · envelope limits · correlation id
│ SMTP Receiver │  250 only once Redis holds the message, otherwise 451
└───────┬───────┘
        │  XADD
        ▼
┌───────────────┐  delivery:messages · group delivery-workers
│  Redis Stream │  an entry stays pending until it is acknowledged
└───────┬───────┘
        │  XREADGROUP
        ▼
┌───────────────┐  DELIVERY_WORKERS goroutines · one entry to one worker
│ Delivery Pool │  a crash leaves the entry pending · XAUTOCLAIM reclaims it
└───────┬───────┘
        │
        ▼
┌───────────────┐  resolve the sender's active policies (one query, cached)
│   Screening   │  recipient-domain and attachment restrictions
└───────┬───────┘  keyword (Aho-Corasick) and regex content rules
        │          resolve one effective action · write an incident
        ▼
┌───────────────┐  DKIM sign · MX lookup · STARTTLS · retry with backoff
│     Relay     │
└───────┬───────┘
        ▼
  Recipient MX
        │
        ▼
      XACK        only after the terminal audit record is written
```

The SMTP session does the minimum: authorize the sender domain, validate the envelope, read the body, mint the correlation id, write a `PROCESSING` audit record, publish. Everything after that belongs to the worker pool.

**`250` means the message crossed into Redis** — nothing more and nothing less. If the publish fails, the receiver completes the audit record as `FAILED` and answers `451`, so the sending MTA retries rather than believing the message was taken.

**An entry is acknowledged only once its outcome is recorded.** A delivery that fails permanently is still a finished message: it gets a terminal `FAILED` audit and is acked, because replaying it would send the mail twice. The single case that leaves an entry pending is an outcome that could not be written down — precisely when reprocessing is safe. A worker that dies mid-message acknowledges nothing, and after `DELIVERY_CLAIM_IDLE` another worker reclaims the entry.

**One correlation id** is minted in the SMTP session before publishing and carried unchanged through the stream, the worker, policy evaluation, the incident, the audit record and any block notice. The Redis entry id, the RFC822 `Message-ID` and the correlation id are three separate things.

Four files tell the whole story:

| File | Role |
|---|---|
| [`handler/smtp/receiver.go`](backend/internal/delivery/handler/smtp/receiver.go) | the SMTP session — validate, publish, answer |
| [`queue.go`](backend/internal/delivery/queue.go) | Redis Streams — publish, consume, ack, reclaim, group creation |
| [`worker.go`](backend/internal/delivery/worker.go) | the pool, the reclaimer, the stats reporter |
| [`service.go`](backend/internal/delivery/service.go) | `DeliveryService.Process` — screening, audit, relay |

An admin configures sending domains, DKIM keys, email users, groups, rules and policies through the dashboard. Policies bind rules and groups together and carry one action: `BLOCK`, `QUARANTINE`, `REDACT` or `AUDIT`. A `BLOCK` withholds recipients; the other three record an incident and deliver.

Each customer also has its own dashboard branding — logo, light/dark theme, language and timezone. See [Custom branding](#custom-branding).

## Repository layout

```
frontend/             Next.js 16 App Router, RTK Query, Tailwind v4, base-ui
backend/
  cmd/api/            one binary: HTTP API + SMTP receiver + delivery workers
  internal/
    auth/             registration, login, JWT cookies, CSRF, identity cache
    middleware/       origin, auth, CSRF, privilege guards
    admin/            configurations, policies, rules, email users, groups, branding
    delivery/
      model.go        EmailMessage, results, the Processor and Sender seams
      queue.go        Redis Streams handoff
      worker.go       worker pool and pending-entry recovery
      service.go      DeliveryService.Process
      handler/smtp/   SMTP server, session, sender authorization
      services/       screening · policy · inspection · restriction
                      adjudication · transmission · recording
      repositories/   provider configs, policy sets, email templates
      dto/            evaluation and policy-set value types
    audit/            delivery audits and email incidents (MongoDB)
    notification/     transactional email from templates
    storage/          S3/MinIO object storage (customer logos)
    config/           environment loading
    db/               GORM models and shared scopes
  migrations/         golang-migrate SQL
  tests/              mirrors internal/, integration tests behind a build tag
  scripts/            send_test_mail.py
  docker-compose.yml  Postgres, Redis, MongoDB, MinIO, Mailpit
```

`admin`, `delivery` and `audit` are peer modules sharing a `handler / services / repositories / dto / utils` layering. `audit` owns its own vocabulary and is reached from `delivery` only through a recorder interface, so it could be extracted as a service later.

Redis carries three unrelated things: the delivery stream, the policy and signing-config caches, and the auth identity cache. They share one client built from `REDIS_URL`.

## Running locally

```bash
cd backend
cp ../.env.example .env
make up
```

Brings up Postgres, Redis, MongoDB, Mailpit and MinIO from `backend/docker-compose.yml` under compose project `dpdp`. Mailpit's UI is on <http://localhost:8025>, MinIO's console on <http://localhost:9001>.

Postgres runs the `pgvector` image — migration `000001` creates the `vector`, `pgcrypto` and `citext` extensions, so a stock `postgres` image will not migrate.

That is infrastructure only. To run the backend from source against those containers:

```bash
make run
```

The consumer group is created at startup (`XGROUP CREATE … MKSTREAM`); an existing group is not an error, so restarts are safe.

To run the backend in Docker too, including a one-shot migration step:

```bash
docker compose --profile app up -d --build
```

The `app` profile expects in-cluster hostnames (`postgres`, `redis`, `mongo`, `minio`, `mailpit`) rather than the `localhost` values in `.env`; the compose file supplies those as overrides, so one env file serves both ways of running.

Frontend:

```bash
cd frontend
npm install
npm run dev
```

<http://localhost:3000>. Set `NEXT_PUBLIC_API_BASE_URL` to `http://localhost:8080/api/v1` — note the `/api/v1` suffix, and note that it is baked in at build time.

## Sending a test message

```bash
cd backend
python3 scripts/send_test_mail.py \
  --host localhost --port 2525 \
  --from you@your-configured-domain \
  --to someone@example.test \
  --body "this is confidential"
```

Standard library only. `--help` lists the rest: multiple recipients, attachments, a fixed `Message-ID`, a pre-existing `DKIM-Signature`, concurrent sends, and full SMTP tracing. The exit code follows the SMTP reply, so `250` and `451` are distinguishable from a script.

Set `RELAY_MX_OVERRIDE=localhost:1025` first so the relay delivers into Mailpit instead of doing a real MX lookup. The envelope sender's domain must be a configured provider configuration; for a policy to apply, the address must also exist as an email user in a group bound to an active policy.

Results land in the dashboard under **Email Protection → Audits**, split into Delivery Audit and Incidents.

To watch the handoff itself:

```bash
redis-cli XLEN delivery:messages                       # entries ever published
redis-cli XPENDING delivery:messages delivery-workers  # taken but not yet acked
```

A healthy idle system reports zero pending. A number that does not fall is a worker that died mid-message; it clears itself once `DELIVERY_CLAIM_IDLE` elapses and another worker reclaims it.

## Custom branding

Branding is customer-level and admin-only. Every customer has exactly one `customer_branding` row:

| Column | Values | Default |
|---|---|---|
| `logo_key` | S3 object key, never a URL | `NULL` |
| `theme` | `LIGHT`, `DARK` | `LIGHT` |
| `language` | `ENGLISH`, `JAPANESE`, `SPANISH` | `ENGLISH` |
| `timezone` | IANA identifier, validated with `time.LoadLocation` | `UTC` |

That row is created in the same transaction as the customer, and migration `000004` backfills every pre-existing customer — so the row is an invariant, not something the code repairs at runtime.

**Reads go through `GET /api/v1/me`**, which every authenticated user may call. There is no branding GET; adding one would mean a second request at dashboard init and a second source of truth. The three write endpoints all require the `admin.branding.edit` privilege and all return `204`:

```
PATCH  /api/v1/admin/branding         theme · language · timezone, each optional
POST   /api/v1/admin/branding/logo    multipart/form-data, logo=<file>
DELETE /api/v1/admin/branding/logo
```

The customer id is never accepted from the request — it comes from the authenticated actor, so an admin cannot reach another customer's branding.

### Identity cache

The `/me` payload is cached in Redis under `auth:identity:<user_id>` for **1 hour**, with `auth:identity:customer:<customer_id>` indexing every cached user of a customer. Branding rides inside that payload, so a branding write invalidates the whole customer's index — not just the acting admin, who would otherwise be the only user to see the change.

The CSRF token is deliberately outside the cache: it is an HMAC of the access-token id and differs per session.

Cache failures degrade rather than fail. A Redis read error falls through to Postgres; an invalidation failure after a committed write logs a warning and still returns success, because the write did happen. The cost is that warm caches serve stale branding until the TTL expires.

### Dashboard language

`next-intl` runs **client-side only**, with no locale routing and no `createNextIntlPlugin`. The locale is not in the URL — it comes from `identity.branding.language` on the `/me` payload, which is only known after authentication, so `IntlProvider` sits inside `AuthProvider` and also drives `<html lang>`.

Catalogues live in `frontend/messages/{en,ja,es}.json` and are keyed by namespace (`common`, `nav`, `table`, `status`, one per feature). Two conventions worth knowing before adding UI:

- **`columns`/`filters` arrays must be built inside the component**, not at module scope, or they cannot reach `useTranslations`.
- **Backend enums are translated through the `status` namespace** by lowercased key, with the raw value as fallback (`t.has(key) ? t(key) : value`). A new enum value renders as-is rather than crashing.

`routes.ts` carries a `labelKey` per node rather than a label; the sidebar and tab strip resolve it against `nav`.

> The Japanese and Spanish catalogues are machine-translated and **have not been reviewed by a native speaker**. This is a security product, and the vocabulary that carries the most risk — authentication, authorization, policy, incident, DLP, enforcement, quarantine, block, recipient, domain — is exactly where machine translation misleads about what the product does. Treat `en.json` as authoritative and get `ja`/`es` reviewed before they reach customers.

The pre-auth pages (login, register, forgot/reset password) are deliberately **not** translated: branding language is customer-level and unknown until `/me` returns, so they have no locale source and would always render in the default.

### Logos

Stored in S3/MinIO at a backend-controlled key — the frontend can never choose the path:

```
customers/{customer_id}/branding/logo/logo.{png|jpg|webp}
```

Uploads must be 1 MB or less and PNG, JPEG or WebP. The type is decided by sniffing the first 512 bytes and then decoding the image, never by the filename or the client's `Content-Type`. The route is additionally wrapped in `http.MaxBytesReader`, since `MaxMultipartMemory` caps buffering rather than request size.

Object storage goes through `aws-sdk-go-v2` with `UsePathStyle`, not `minio-go`. That is deliberate: `minio-go` rejects any endpoint carrying a path (`Endpoint url cannot have fully qualified paths`), and it signs before its `Transport` runs, so a path-rewriting `RoundTripper` would produce `SignatureDoesNotMatch`. Supabase Storage's S3 endpoint always includes `/storage/v1/s3`, so a path-capable client is required. AWS S3, Cloudflare R2, Wasabi and local MinIO all work through the same client.

The database stores `logo_key`; the presigned GET URL is minted when the branding snapshot is built and cached for an hour with it. Nothing that expires is ever persisted — the invariant is `S3_LOGO_PRESIGN_TTL > IdentityTTL`, so a served URL always outlives the cache entry carrying it.

Replacement order is upload → update `logo_key` → commit → invalidate cache → delete the old object. A failure at any step leaves the old logo working; the worst case is an orphaned object.

## Migrations

```bash
make migrate-up
make migrate-down
make migrate-create name=add_something
```

## Tests

```bash
make test-unit            # no infrastructure required
make test-db              # drops, creates and migrates the test database
make test-integration     # needs Postgres, Redis, MongoDB, Mailpit, MinIO
make test                 # both
```

Integration tests sit behind a `//go:build integration` tag, so `make test-unit` compiles neither them nor their helpers. Run `go build -tags integration ./tests/...` after changing shared wiring, or a break there stays invisible until someone runs the full suite.

They need:

```
TEST_DATABASE_URL      postgres://...
TEST_REDIS_URL         redis://localhost:6379/1   non-zero database, it is flushed
TEST_MONGO_URI         mongodb://localhost:27017
TEST_MONGO_DATABASE    secureplus_test
TEST_MAILPIT_ADDR      localhost:1025
TEST_S3_ENDPOINT       localhost:9000            optional, logo tests skip without it
TEST_S3_ACCESS_KEY     minioadmin
TEST_S3_SECRET_KEY     minioadmin
TEST_S3_BUCKET         dpdp-test                 must differ from S3_BUCKET
TEST_S3_USE_SSL        false
```

`TEST_REDIS_URL` must select a different logical database than `REDIS_URL`; the suite refuses to run otherwise, because it flushes what it connects to. Each delivery test also uses its own stream name, so runs do not interfere.

`TEST_S3_*` is the only optional group. Unset or unreachable object storage skips the logo round-trip tests and leaves the rest running — the bucket is created on connect, so it need not exist beforehand.

`make` loads `.env` itself; a bare `go test` does not. To run a subset directly:

```bash
set -a && . ./.env && set +a
go test -tags integration -p 1 -count=1 ./tests/delivery/ -run TestQueue -v
```

## Configuration

Everything is environment-driven. The ones without defaults:

| Variable | Notes |
|---|---|
| `DATABASE_URL` | an empty value silently falls back to a local Unix socket |
| `REDIS_URL` | full URL — `rediss://` enables TLS. One client serves the delivery stream, the caches and the identity cache |
| `MONGO_URI`, `MONGO_DATABASE` | the process exits if Mongo is unreachable |
| `PASSWORD_PEPPER`, `CSRF_SECRET` | mixed into every password hash and CSRF token |
| `CORS_ALLOWED_ORIGIN` | exact origin, no trailing slash; credentials forbid wildcards |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` | object storage for customer logos; the process exits if it is unreachable. The endpoint may include a scheme and a path — `https://<ref>.storage.supabase.co/storage/v1/s3` works, as does a bare `localhost:9000` |

Notable defaults:

| Variable | Default | Notes |
|---|---|---|
| `S3_USE_SSL` | `false` | only consulted when `S3_ENDPOINT` has no scheme; **must be `false` for local MinIO**, which serves plain HTTP |
| `S3_REGION` | `us-east-1` | required by SigV4 even on providers that ignore it |
| `S3_LOGO_PRESIGN_TTL` | `24h` | must exceed the 1 h identity cache TTL, or a cached logo URL can outlive its signature; values below that are ignored |
| `APP_PORT` | — | no default; an unset value binds a random port |
| `APP_ENV` | — | anything but `production` logs every SQL statement |
| `COOKIE_SECURE` / `COOKIE_SAMESITE` | `false` / `lax` | cross-site deployments need `true` / `none` |
| `SMTP_SERVER_ADDR` | `:2525` | Docker maps `25:2525` so the process stays unprivileged |
| `SMTP_MAX_SIZE` | 10 MB | the only message-size limit; nothing larger can reach the stream |
| `SMTP_MAX_CONNECTIONS` | 100 | SMTP connection capacity, independent of delivery concurrency |
| `DELIVERY_WORKERS` | 4 | delivery-processing concurrency, per instance |
| `DELIVERY_STREAM` | `delivery:messages` | the handoff stream |
| `DELIVERY_CONSUMER_GROUP` | `delivery-workers` | shared across instances, so one entry goes to one worker |
| `DELIVERY_CLAIM_IDLE` | `5m` | pending time after which another worker reclaims an entry |
| `DELIVERY_BATCH_SIZE` / `DELIVERY_BLOCK_TIME` | `10` / `1s` | `XREADGROUP` count and block duration |
| `RELAY_MX_OVERRIDE` | empty | force all mail to one host, for development |
| `RELAY_MAX_ATTEMPTS` | `3` | with 1m → 5m backoff, capped at 15m |
| `SHUTDOWN_TIMEOUT` | `30s` | bounds the worker drain |

The remaining `SMTP_*`, `RELAY_*` and `DELIVERY_*` knobs live in `internal/config/smtpserver.go`, `internal/config/relay.go` and `internal/config/delivery.go`.

`SMTP_MAX_CONNECTIONS` and `DELIVERY_WORKERS` are independent: the first caps concurrent SMTP conversations, the second caps concurrent deliveries, and the stream absorbs the difference. Scaling out adds workers to the same consumer group, so throughput rises without duplicating deliveries; caches and the worker count stay per-instance.

On shutdown the SMTP listener stops first, then the HTTP API, then the workers stop asking Redis for new entries while finishing the message they hold. Anything unfinished when the budget expires is simply never acknowledged, so the next process to start reclaims it.
