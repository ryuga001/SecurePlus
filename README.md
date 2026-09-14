# SecurePlus

An outbound email security gateway. Mail from a tenant's mail system is accepted over SMTP, evaluated against that tenant's data-loss-prevention policies, DKIM-signed, relayed to the recipient's MX, and recorded — with a dashboard over the resulting delivery audits and policy incidents.

## What it does

```
Exchange / Gmail
       │  SMTP
       ▼
┌──────────────┐   authorize sender domain · envelope limits · correlation id
│ SMTP Receiver│
└──────┬───────┘
       │ 250 accepted, processing continues in the background
       ▼
┌──────────────┐   resolve the sender's active policies (one query, cached)
│ Policy Engine│   recipient-domain + attachment restrictions
└──────┬───────┘   keyword (Aho-Corasick) and regex content rules
       │           resolve one effective action · write an incident
       ▼
┌──────────────┐   DKIM sign · MX lookup · STARTTLS · retry with backoff
│    Relay     │
└──────┬───────┘
       ▼
Recipient MX

Receiver ─┐
Engine  ──┼──▶ MongoDB   delivery_audits   PROCESSING → SUCCESS | FAILED
Relay   ──┘                email_incidents  one per flagged message
```

An admin configures sending domains, DKIM keys, email users, groups, rules and policies through the dashboard. Policies bind rules and groups together and carry one action: `BLOCK`, `QUARANTINE`, `REDACT` or `AUDIT`.

Each customer also has its own dashboard branding — logo, light/dark theme, language and timezone. See [Custom branding](#custom-branding).

## Repository layout

```
frontend/           Next.js 16 App Router, RTK Query, Tailwind v4, base-ui
backend/
  cmd/api/          one binary: HTTP API + SMTP receiver
  internal/
    auth/           registration, login, JWT cookies, CSRF, identity cache
    middleware/     origin, auth, CSRF, privilege guards
    admin/          configurations, policies, rules, email users, groups, branding
    delivery/       receiver → engine → relay  (the mail path)
      engine/       policy evaluation, matchers, action triggers
    audit/          delivery audits and email incidents (MongoDB)
    notification/   transactional email from templates
    storage/        S3/MinIO object storage (customer logos)
    config/         environment loading
    db/             GORM models and shared scopes
  migrations/       golang-migrate SQL
  tests/            mirrors internal/, integration tests behind a build tag
  scripts/          send_test_mail.py
  docker-compose.yml  Postgres, Redis, MongoDB, MinIO, Mailpit
```

`admin`, `delivery` and `audit` are peer modules, each with its own `handler / services / repositories / dto / utils` layering. `audit` defines its own vocabulary and is not imported by `delivery` except through a recorder interface, so it could be extracted as a service later.

## Running locally

```bash
cd backend
cp ../.env.example .env
make up
```

Brings up Postgres, Redis, MongoDB, Mailpit and MinIO from `backend/docker-compose.yml`, under compose project `dpdp`. Mailpit's UI is on <http://localhost:8025>, MinIO's console on <http://localhost:9001>.

Postgres runs the `pgvector` image — migration `000001` creates the `vector`, `pgcrypto` and `citext` extensions, so a stock `postgres` image will not migrate.

That brings up infrastructure only. To run the backend itself from source against those containers:

```bash
make run
```

To run the backend in Docker too, including a one-shot migration step:

```bash
docker compose --profile app up -d --build
```

The `app` profile expects in-cluster hostnames (`postgres`, `redis`, `mongo`, `minio`, `mailpit`) rather than the `localhost` values in `.env`; the compose file supplies those as overrides on top of `.env`, so one env file serves both ways of running.

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

Standard library only. `--help` lists the rest: multiple recipients, attachments, a fixed `Message-ID`, a pre-existing `DKIM-Signature`, concurrent sends, and full SMTP tracing.

Set `RELAY_MX_OVERRIDE=localhost:1025` first so the relay delivers into Mailpit instead of doing a real MX lookup. The envelope sender's domain must be a configured provider configuration, and for a policy to apply the address must also exist as an email user in a group bound to an active policy.

Results land in the dashboard under **Email Protection → Audits**, split into Delivery Audit and Incidents.

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
make test-db              # creates and migrates the test database
make test-integration     # needs Postgres, Redis, MongoDB, Mailpit, MinIO
make test                 # both
```

Integration tests sit behind a `//go:build integration` tag and need:

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

`TEST_REDIS_URL` must select a different logical database than `REDIS_URL`; the suite refuses to run otherwise, because it flushes what it connects to.

`TEST_S3_*` is the only optional group. Unset or unreachable object storage skips the logo round-trip tests and leaves the rest running — the bucket is created on connect, so it need not exist beforehand.

Note that `make` loads `.env` itself; a bare `go test` does not. To run a subset directly:

```bash
set -a && . ./.env && set +a
go test -tags integration -p 1 -count=1 ./tests/admin/handler/ -run TestBranding -v
```

## Configuration

Everything is environment-driven. The ones without defaults:

| Variable | Notes |
|---|---|
| `DATABASE_URL` | an empty value silently falls back to a local Unix socket |
| `REDIS_URL` | full URL — `rediss://` enables TLS |
| `MONGO_URI`, `MONGO_DATABASE` | the process exits if Mongo is unreachable |
| `PASSWORD_PEPPER`, `CSRF_SECRET` | mixed into every password hash and CSRF token |
| `CORS_ALLOWED_ORIGIN` | exact origin, no trailing slash; credentials forbid wildcards |
| `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET` | object storage for customer logos; the process exits if it is unreachable |

Notable defaults:

| Variable | Default | Notes |
|---|---|---|
| `S3_USE_SSL` | `false` | **must be `false` for local MinIO**, which serves plain HTTP — otherwise startup fails with `server gave HTTP response to HTTPS client`. `S3_ENDPOINT` is a bare `host:port`, no scheme |
| `S3_LOGO_PRESIGN_TTL` | `24h` | must exceed the 1 h identity cache TTL, or a cached logo URL can outlive its signature; values below that are ignored |
| `APP_PORT` | — | no default; an unset value binds a random port |
| `APP_ENV` | — | anything but `production` logs every SQL statement |
| `COOKIE_SECURE` / `COOKIE_SAMESITE` | `false` / `lax` | cross-site deployments need `true` / `none` |
| `SMTP_SERVER_ADDR` | `:2525` | Docker maps `25:2525` so the process stays unprivileged |
| `SMTP_MAX_SIZE` / `SMTP_MAX_DELIVERIES` | 10 MB / 32 | messages are held in memory; the product of these is the ceiling |
| `RELAY_MX_OVERRIDE` | empty | force all mail to one host, for development |
| `RELAY_MAX_ATTEMPTS` | 3 | with 1m → 5m backoff, capped at 15m |

The remaining `SMTP_*` and `RELAY_*` knobs are in `internal/config/smtpserver.go` and `internal/config/relay.go`.

## Deployment

The frontend deploys anywhere. **The backend cannot run on an HTTP-only platform** — Render, Vercel, Heroku and Railway publish a single HTTPS entrypoint and no raw TCP port, so the SMTP receiver binds successfully inside the container and is unreachable from outside. Mail senders find you by MX record, which implies port 25, and an Exchange Online outbound connector has no port field at all.

What works:

- **A VM with a public IP** — the whole compose stack, port 25 both directions, and a PTR record, which is what large providers actually check. Note most cloud providers block *outbound* 25 by default; inbound is generally open.
- **A platform with raw TCP support**, with a dedicated IPv4 address.
- **Split hosting** — the dashboard and API on a PaaS, the SMTP receiver on a small VM, both pointed at the same databases. No code change; both listeners already start in one binary.

If the API and dashboard are on different registrable domains, set `COOKIE_SAMESITE=none` and `COOKIE_SECURE=true` or the browser will discard the auth cookies after login.

## Current limitations

- **`QUARANTINE` and `REDACT` are not implemented.** Both resolve an action, write an incident and invoke a stub executor; the message still relays. `BLOCK` sends the sender a notice from `no-reply@<their domain>` using a database template.
- **Only recipient removal is enforced.** A recipient whose domain violates a restriction is dropped from the envelope and recorded as `BLOCKED`; the message goes to everyone else.
- **No bounces.** A `250` is a promise of custody, and a delivery that ultimately fails is recorded in the audit and nowhere else.
- **The receiver authenticates nothing.** Authorization is by envelope sender domain, which is forgeable, so the port must not be publicly reachable without an IP allowlist in front of it.
- **In-process retries do not survive a restart.** A message in retry backoff when the process stops is left `PROCESSING` with no sweeper to reconcile it.
- **DKIM private keys are stored unencrypted** in Postgres and read on every policy-cache miss.
- Sender and recipient addresses persist in MongoDB indefinitely. Matched content is deliberately never stored — incidents record the configured rule and an occurrence count, never the text that matched.
