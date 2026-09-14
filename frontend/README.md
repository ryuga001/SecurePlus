# SecurePlus Frontend

The admin dashboard: email provider configurations, policies, rules, users and groups, plus delivery audits and policy incidents.

## Development

```bash
cp .env.example .env.local
npm install
npm run dev
```

<http://localhost:3000>, against a backend on <http://localhost:8080>.

## Configuration

```
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
```

The `/api/v1` suffix is part of the value. `NEXT_PUBLIC_*` variables are inlined at build time, so changing this requires a rebuild, not just a restart.

The backend must allow this origin through `CORS_ALLOWED_ORIGIN`, and auth uses cookies with `credentials: "include"` — so if the two are deployed on different registrable domains, the backend also needs `COOKIE_SAMESITE=none` and `COOKIE_SECURE=true`.

## Stack

- Next.js 16, App Router, TypeScript
- Redux Toolkit Query — one slice per resource in `src/store/api/`
- Tailwind CSS v4, base-ui primitives in `src/components/ui/`

## Conventions

- `src/lib/routes.ts` is the single source of truth for navigation. A node with a `routePath` and `children` renders as one sidebar link whose children become page tabs.
- `src/components/data-table/` is a config-driven table: pass an RTK Query hook, column and filter configs, row actions, and it handles paging, sorting, debounced filtering and server-side queries.
