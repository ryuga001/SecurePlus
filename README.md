# SecurePlus

An outbound email security gateway. Mail from a tenant's mail system is accepted over SMTP, queued, evaluated against that tenant's data-loss-prevention policies, DKIM-signed, relayed to the recipient's MX, and recorded — with a dashboard over the resulting delivery audits and policy incidents.

It also finds personal data at rest. [Data discovery](#data-discovery) scans a tenant's cloud storage — Azure Blob, AWS S3, Google Drive, SharePoint and OneDrive — with the same keyword and regex rules, and records what each file contains.

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
  cmd/api/            one binary: HTTP API + SMTP receiver + delivery workers + discovery scanner
  internal/
    auth/             registration, login, JWT cookies, CSRF, identity cache
    middleware/       origin, auth, CSRF, privilege guards
    admin/            configurations, policies, rules, email users, groups, branding,
                      data discovery configurations, policies and scans
    datadiscovery/
      provider/       one file per source: listing, streaming download, target browsing
      strategy/       per-source validation, Connect, per-configuration BrowseTargets
      scanner/        scan lifecycle, file queue, workers, processors, evaluator
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
make up
```

Create `backend/.env` first with the variables under [Configuration](#configuration). There is no committed template (`.env.example` is git-ignored).

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

## Profile

The profile page edits four fields: the organization name, which is customer-level, and the signed-in user's first name, last name and admin email.

```
GET    /api/v1/admin/profile   → 200 {org_name, first_name, last_name, admin_email}
PATCH  /api/v1/admin/profile   {org_name?, first_name?, last_name?} → 200 with the updated profile
```

- Any authenticated user can read their profile and change their own name.
- Changing `org_name` requires `admin.organization.edit` (migration 000009, granted to `admin` roles). Without it the request returns `403 forbidden` and nothing is written, including any name change in the same request.
- Resending an unchanged `org_name` does not need the privilege.
- `admin_email` is the sign-in identity. It is returned but never written: sending it returns `400 admin_email_immutable`. Changing it safely needs an ownership check on the new address first.
- Values are trimmed and inner whitespace is collapsed:
  - `org_name` must be 2–100 characters, and unique ignoring case; a clash returns `409 org_name_taken`.
  - Names must be 1–50 characters.
- The whole update runs in one transaction.
- A name change drops only the caller's cached `/me`. An org rename drops the cached `/me` of every user in that customer, because the sidebar shows the org name.

## Admin Console: roles and users

Admins can create custom roles (named sets of privileges) and add dashboard users, each with one role. This reuses the existing `roles`, `privileges` and `role_privileges` tables and the existing `RequirePrivilege` checks. There is no second permission system.

**Who can use it.** Every route below is behind `RequireAdminRole`: the caller's role must have type `admin`. Adding a privilege for these routes was ruled out on purpose, so a custom role can never manage roles or users.

**Roles a tenant sees and can assign:**
- the shared system Admin role (customer 1, type `admin`), which every registered tenant already uses;
- the tenant's own roles of type `custom`.

`Super Admin` is never exposed. The system Admin role is read-only: `syncPrivileges` keeps it at every privilege, and editing it would change every tenant.

```
GET    /api/v1/admin/privileges          every DASHBOARD privilege name
GET    /api/v1/admin/roles               list, with up to 3 privilege and user previews plus totals
GET    /api/v1/admin/roles/options       id, name, system: for the user role picker and filter
GET    /api/v1/admin/roles/:id
POST   /api/v1/admin/roles               {name, description?, privileges: [names]}
PUT    /api/v1/admin/roles/:id           same body; replaces the whole privilege set
DELETE /api/v1/admin/roles/:id
GET    /api/v1/admin/users?search=&role_id=
GET    /api/v1/admin/users/:id
POST   /api/v1/admin/users               {first_name, last_name, email, role_id} → {user, invitation_sent}
PUT    /api/v1/admin/users/:id           {first_name, last_name, role_id}
DELETE /api/v1/admin/users/:id
```

**Rules the backend enforces:**
- **Privileges** are identified by name, not numeric id. An unknown or non-DASHBOARD name rejects the whole request with `400 privilege_not_found`.
- **Role names** keep the existing case-sensitive `UNIQUE(customer_id, name)`. A clash, or a name matching the system `Admin` role ignoring case, returns `409 role_name_taken`.
- **Writing to a system role** returns `409 system_role_immutable`.
- **A role with users can't be deleted** (`409 role_in_use`). Migration `000010` changed `dashboard_users.role_id` from `ON DELETE SET NULL` to `NO ACTION`, so the database refuses to orphan users even under a race.
- **`role_id`** must be the system Admin role or one of the tenant's custom roles. Anything else (Super Admin, another tenant's role, a missing id) returns `400 role_not_found`.
- **Admins can't change their own role or delete themselves** (`409 self_modification`). A tenant always keeps at least one system-Admin user (`409 last_administrator`), checked under row locks.
- **Email:** new users get the `user_invited_credentials` email with a random temporary password. If sending fails, the user is still created and the response says `invitation_sent: false`, so they can use Forgot password instead. The email is the sign-in identity and can't be changed later.

**When changes take effect:**
- **Editing a custom role's privileges** drops that role's `priv:role:<id>` cache, so the change applies on the users' next request. The system Admin role's cache is never touched.
- **Changing a user's role** is eventually consistent. It applies at their next token refresh (at most `ACCESS_TTL`, 15 minutes by default) and does not log them out.
- **Deleting a user** bumps their token version, so their current sessions end immediately.

## Sessions

Login sets three cookies:

| Cookie | Holds | Lifetime | Notes |
|---|---|---|---|
| `dpdp_at` | access JWT | `ACCESS_TTL` (15 m) | HttpOnly |
| `dpdp_rt` | refresh JWT | `REFRESH_TTL` (30 days) | HttpOnly, path `/api/v1/auth` only, SameSite Strict |
| `dpdp_csrf` | CSRF token | 30 days | readable by the page |

Mutating requests send `X-CSRF-Token`. On authenticated routes it must equal an HMAC of the access token's id, so it changes on every refresh. On `POST /auth/refresh` it must equal the `dpdp_csrf` cookie.

**Refresh tokens rotate.** Each refresh blacklists the token it used. A second refresh with the same token within 10 seconds (parallel tabs) receives the same new pair. After that, a reused token is treated as stolen: the user's token version is bumped and every session ends.

**The frontend refreshes silently.** Both API clients — `lib/api.ts` and RTK Query's `baseQuery` in `store/api/base-api.ts` — handle expiry the same way:

- A 401 triggers one shared refresh, and the request is retried once.
- A `csrf_failed` 403, which means another tab refreshed, re-reads the token from `/me` and retries.
- After a reload the CSRF token is gone from memory, so the client mints one through `/auth/csrf` before refreshing.

Only a rejected refresh token signs the user out. A backend outage (503) keeps the session, and the refresh endpoint clears cookies only for invalid or replayed tokens.

## Data discovery

An admin connects a cloud account (a **configuration**, whose secret is encrypted with `DATA_DISCOVERY_MASTER_KEY`). They then say where to look and what to look for (a **policy**: targets, file types, rules) and run a **scan**. Every supported file is streamed through the policy's rules, and each file's matches are recorded. The record holds counts and byte offsets per rule, never the matched text.

```
POST /admin/data-discovery/scans
        │  scan row PENDING · one active scan per policy
        ▼
┌───────────────┐  claim the oldest PENDING row · FOR UPDATE SKIP LOCKED
│    Scanner    │  one scan per process · wakes on create, polls every 30s
└───────┬───────┘  load policy · decrypt credential · compile rules · connect
        │                                                PENDING → RUNNING
        ▼
┌───────────────┐  targets strictly one after another
│   Producer    │  Source.List → file-type filter → FileQueue.Put
└───────┬───────┘  blocks while QUEUE_CAPACITY files are waiting
        ▼
┌───────────────┐  FILE_WORKERS goroutines per target, one file each
│ File workers  │  Source.Open → processor → evaluator → upsert result
└───────┬───────┘
        ▼
  next target … then COMPLETED · PARTIAL · FAILED
```

**The scan row is the job.** There is no queue service:

- A PENDING row is pending work, and `started_at` marks it claimed.
- Every status change is a conditional `UPDATE … WHERE status = …`, so a scan is finalized exactly once.
- Counters are rewritten every 5 seconds, which doubles as a heartbeat. A RUNNING scan silent for 2 minutes belonged to a crashed process, and the next sweep marks it `FAILED` / `INTERRUPTED`. A graceful shutdown writes that status itself.

**Targets run one after another; files within a target run in parallel.** A target is finished only when all of these hold:

- its listing has ended;
- the queue is drained;
- every worker has saved its last file.

**The queue holds waiting files only**, and only as metadata, never content. A worker taking a file frees that slot immediately, so `FILE_WORKERS` alone caps how many files are open at once. Files finish in any order.

**Results are upserted per file** on `(scan_id, file_key)`. Nothing accumulates in memory, and reprocessing a file is idempotent.

**Counters are exact.** For every completed or partial scan:

- `discovered = supported + skipped`
- `supported = processed = succeeded + failed`

Skipped files (no processor, or not in the policy's file types) never make a scan PARTIAL.

**One bad file doesn't stop a scan.**

| Failure | Effect |
|---|---|
| A single file (`FETCH_NOT_FOUND`, `PARSE_FAILED`, `TIMEOUT`, `LIMIT_EXCEEDED`, …) | That file is recorded FAILED and its worker moves on |
| A target's listing | That target fails; the next target still runs |
| Scan level: credential, rules, database, cancellation | The whole scan fails |

**Logs never carry file names or content.** Files are identified by `file_ref`, a truncated SHA-256 of the key.

### Sources

| Source | Target, as stored | Available targets lists | Grant |
|---|---|---|---|
| Azure Blob | `container/prefix` | containers | Storage Blob Data Reader on the account |
| AWS S3 | `bucket/prefix` | buckets, with region | `s3:ListAllMyBuckets`, `s3:ListBucket`, `s3:GetObject`, `sts:GetCallerIdentity` |
| SharePoint | `/sites/x/Library//folder`, or a full site URL | sites, then a site's libraries | Graph application `Sites.Read.All`, `Files.Read.All` |
| OneDrive | `user@domain//folder` | users | Graph application `Files.Read.All`, `User.Read.All` |
| Google Drive | shared drive name or ID, `user@domain` or `My Drive`, then `//folder` | shared drives, then Workspace users | Drive API, `drive.readonly`; see below for user drives |

**Google user drives** need extra setup:

- Domain-wide delegation for the service account, with both `drive.readonly` and `admin.directory.user.readonly`.
- The Admin SDK API enabled in the service account's Cloud project.
- An admin as the configuration's subject. User listing impersonates that admin; each user's drive is scanned by impersonating that user.
- Files shared with a user but not in their My Drive aren't included.
- Google Docs, Sheets and Slides are exported as DOCX, XLSX and PPTX.

**How sources behave:**

- **Case:** bucket and container names are lowercased, but prefixes keep their case because object keys are case-sensitive.
- **Folder walking:** SharePoint, OneDrive and Drive are walked depth-first. Memory is bounded by depth × page size. Folders deeper than 64 levels are skipped with a warning.
- **Tokens:** cached and refreshed 5 minutes before expiry.
- **Retries:** 429 and 503 responses are retried, honouring Retry-After.

The policy form's **Available targets** picker calls `GET /admin/data-discovery/configurations/:id/targets`:

- Search is done server-side, pages come from a cursor, and the list scrolls infinitely.
- The All / Selected filter shows what is already picked.
- Each cursor is sealed with the credential secret box and bound to the tenant, configuration, source and search. This matters because Graph cursors are URLs that the stored credential would follow; a tampered one is rejected.

### File formats

| Formats | Handling |
|---|---|
| txt, csv, json, xml, js | streamed; UTF-8 and UTF-16 BOMs detected |
| docx, xlsx, pptx | spooled to `DATA_DISCOVERY_SPOOL_DIR` (ZIP needs random access), then each XML part is inflated and stripped as a stream; 2 GiB decompressed cap against zip bombs |
| pdf | spooled (128 MB max), parsed one page at a time; image-only PDFs have no text |
| doc, xls, ppt, archives, images, executables | skipped and counted |

A policy's file types match by extension. A name without one falls back to the provider's MIME type. An empty list means every supported format.

### Matching

Rules compile once per scan through the email engine's compiler (`delivery/services/policy`), so a keyword or regex means the same thing in both products.

- **Keywords** are NFKC-normalized and lowercased, then matched in one Aho-Corasick pass whose state carries across chunks. Matches that span chunk boundaries are found without re-reading anything.
- **Regexes** (RE2) run over 1 MiB windows that overlap by 4 KiB, and a match counts only where it starts. Matches up to 4 KiB long are therefore counted exactly once.
- **Evaluation concurrency:** `DATA_DISCOVERY_EVALUATOR_WORKERS` goroutines evaluate rule shards, shared by every file worker.
- **Size limits:** a policy whose keywords exceed 64 KiB, or whose regexes compile past 64k instructions, fails with `RULES_TOO_LARGE`. That limit keeps rule state inside the memory budget.
- **Speed:** Go's RE2 does roughly 15–40 MB/s per regex per core on patterns without a literal prefix, so large regex sets are CPU-bound.

### Memory

Scanner memory grows with concurrency, not with file size.

| Contributor | Worst case |
|---|---|
| Shared state: automaton, compiled regexes, one listing page, queue | about 40 MB |
| Each active file: download buffers, extraction, evaluation window | about 10 MB more |
| An active PDF, instead of the per-file figure above | up to ~32 MB for a 128 MB file (the parser holds the xref table and a page) |

Plan for the case where every worker has a PDF open at once.

Spooled files live on disk, up to `FILE_WORKERS × MAX_SPOOL_BYTES`.

A progress line is logged every 5 seconds with `heap_inuse_mb`, `files_active` and `queue_pending`. For a hard ceiling, set `GOMEMLIMIT`; it applies to the whole process, API included.

### Endpoints

```
POST   /api/v1/admin/data-discovery/scans                      admin.discovery.scan.create   {"policy_id": N} → 202
GET    /api/v1/admin/data-discovery/scans                      admin.discovery.scan.view     ?policy_id=&status=
GET    /api/v1/admin/data-discovery/scans/:id                  admin.discovery.scan.view     counters and targets
GET    /api/v1/admin/data-discovery/scans/:id/files            admin.discovery.scan.view     ?status=&with_findings=true
GET    /api/v1/admin/data-discovery/configurations/:id/targets admin.discovery.policy.view   ?source_type=&search=&cursor=&parent=
```

A second scan for a policy that already has one PENDING or RUNNING returns `409`. Configurations and policies have the usual CRUD under `/admin/data-discovery/configurations` and `/admin/data-discovery/policies`.

In the dashboard, **Data Discovery → Scans** lists scans and refreshes them while they run. Clicking a scan opens its counters and targets, and from there its per-file results, each with the rules it matched.

### Where the code is

| Path | Role |
|---|---|
| [`scanner/scanner.go`](backend/internal/datadiscovery/scanner/scanner.go) | claim loop, `Receive` → preProcess / process / postProcess, terminal status |
| [`scanner/pipeline.go`](backend/internal/datadiscovery/scanner/pipeline.go) | per-target producer and file workers, counters, progress |
| [`scanner/queue.go`](backend/internal/datadiscovery/scanner/queue.go) | the bounded queue of waiting files |
| [`scanner/evaluator.go`](backend/internal/datadiscovery/scanner/evaluator.go) | windowed keyword and regex evaluation |
| [`scanner/processor.go`](backend/internal/datadiscovery/scanner/processor.go), `office.go`, `pdf.go` | format processors and spooling |
| [`provider/`](backend/internal/datadiscovery/provider/) | per-source listing, download, target browsing, token handling |
| [`strategy/`](backend/internal/datadiscovery/strategy/) | per-source `Connect`, per-configuration `BrowseTargets` |
| [`services/datadiscovery/scan_service.go`](backend/internal/admin/services/datadiscovery/scan_service.go) | the scan API, and the store the scanner writes through |

## Migrations

```bash
make migrate-up
make migrate-down
make migrate-create name=add_something
```

Scans need migration `000008_data_discovery_scans`. It adds the scan, target and file-result tables, and the `admin.discovery.scan.*` privileges for admin roles. Role privileges are cached in Redis for 10 minutes, so signed-in users see the new privileges once that expires and they reload the page.

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

Data discovery tests need no network or cloud accounts. Provider tests drive fake HTTP backends (including the real AWS SDK), and pipeline tests use a fake source and store:

```bash
go test ./tests/datadiscovery/...                        # includes 128 MiB streaming memory tests
go test -short -race ./tests/datadiscovery/scanner/      # skips the memory tests, checks the workers for races
go test -tags integration ./tests/datadiscovery/scans/   # needs TEST_DATABASE_URL
```

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
| `DATA_DISCOVERY_MASTER_KEY` | 32 random bytes, base64 (`openssl rand -base64 32`); encrypts every data discovery credential, and the process exits without it. Losing it makes stored credentials unreadable |

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
| `ACCESS_TTL` / `REFRESH_TTL` | `15m` / `720h` | see [Sessions](#sessions) |
| `DATA_DISCOVERY_SCANNER_ENABLED` | `true` | turn off to run an API-only instance that never claims scans |
| `DATA_DISCOVERY_FILE_WORKERS` | `5` | 1–8; files processed at once within a target |
| `DATA_DISCOVERY_QUEUE_CAPACITY` | `100` | 1–5000; files waiting for a worker |
| `DATA_DISCOVERY_EVALUATOR_WORKERS` | `4` | 1–16; rule-evaluation goroutines shared by all file workers |
| `DATA_DISCOVERY_CHUNK_BYTES` | 1 MiB | 64 KiB–8 MiB; evaluation window size |
| `DATA_DISCOVERY_FILE_TIMEOUT` | `30m` | per file; a slow file fails with `TIMEOUT` and the scan moves on |
| `DATA_DISCOVERY_SPOOL_DIR` | OS temp dir | must exist; holds office and PDF files while they are parsed |
| `DATA_DISCOVERY_MAX_SPOOL_BYTES` | 1 GiB | larger office and PDF files fail with `FILE_TOO_LARGE` |
| `DATA_DISCOVERY_TEST_TIMEOUT` | `20s` | connection tests and target browsing; also the response-header timeout for scan downloads |
| `DATA_DISCOVERY_KEY_VERSION` | `1` | bumped when the master key is rotated |

The remaining `SMTP_*`, `RELAY_*` and `DELIVERY_*` knobs live in `internal/config/smtpserver.go`, `internal/config/relay.go` and `internal/config/delivery.go`.

`SMTP_MAX_CONNECTIONS` and `DELIVERY_WORKERS` are independent: the first caps concurrent SMTP conversations, the second caps concurrent deliveries, and the stream absorbs the difference. Scaling out adds workers to the same consumer group, so throughput rises without duplicating deliveries; caches and the worker count stay per-instance.

On shutdown the SMTP listener stops first, then the HTTP API, then the workers stop asking Redis for new entries while finishing the message they hold. Anything unfinished when the budget expires is simply never acknowledged, so the next process to start reclaims it.
