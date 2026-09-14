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

## Repository layout

```
frontend/           Next.js 16 App Router, RTK Query, Tailwind v4, base-ui
backend/
  cmd/api/          one binary: HTTP API + SMTP receiver
  internal/
    auth/           registration, login, JWT cookies, CSRF
    middleware/     origin, auth, CSRF, privilege guards
    admin/          configurations, policies, rules, email users, groups
    delivery/       receiver → engine → relay  (the mail path)
      engine/       policy evaluation, matchers, action triggers
    audit/          delivery audits and email incidents (MongoDB)
    notification/   transactional email from templates
    config/         environment loading
    db/             GORM models and shared scopes
  migrations/       golang-migrate SQL
  tests/            mirrors internal/, integration tests behind a build tag
  scripts/          send_test_mail.py
```

`admin`, `delivery` and `audit` are peer modules, each with its own `handler / services / repositories / dto / utils` layering. `audit` defines its own vocabulary and is not imported by `delivery` except through a recorder interface, so it could be extracted as a service later.

## Running locally

```bash
cd backend
cp ../.env.example .env
make up
```

Brings up Postgres, Redis, MongoDB, Mailpit and MinIO, runs migrations, then the backend. Mailpit's UI is on <http://localhost:8025>.

To run the backend from source against those containers:

```bash
make run
```

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
make test-integration     # needs Postgres, Redis, MongoDB, Mailpit
make test                 # both
```

Integration tests sit behind a `//go:build integration` tag and need:

```
TEST_DATABASE_URL      postgres://...
TEST_REDIS_URL         redis://localhost:6379/1   non-zero database, it is flushed
TEST_MONGO_URI         mongodb://localhost:27017
TEST_MONGO_DATABASE    secureplus_test
TEST_MAILPIT_ADDR      localhost:1025
```

`TEST_REDIS_URL` must select a different logical database than `REDIS_URL`; the suite refuses to run otherwise, because it flushes what it connects to.

## Configuration

Everything is environment-driven. The ones without defaults:

| Variable | Notes |
|---|---|
| `DATABASE_URL` | an empty value silently falls back to a local Unix socket |
| `REDIS_URL` | full URL — `rediss://` enables TLS |
| `MONGO_URI`, `MONGO_DATABASE` | the process exits if Mongo is unreachable |
| `PASSWORD_PEPPER`, `CSRF_SECRET` | mixed into every password hash and CSRF token |
| `CORS_ALLOWED_ORIGIN` | exact origin, no trailing slash; credentials forbid wildcards |

Notable defaults:

| Variable | Default | Notes |
|---|---|---|
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
