# DPDP

Monorepo.

## Structure

- `frontend/` — Next.js + shadcn/ui (own `.env.local`)
- `backend/` — Go + Gin API, `docker-compose.yml` and `.env`

## Backend

```bash
cd backend
cp .env.example .env
docker compose up --build
```

Starts Postgres (pgvector), Redis, Mailpit, runs migrations, then the API on `APP_PORT` (default `8080`). Mailpit UI on `MAILPIT_UI_PORT` (default `8025`).

Migrations:

```bash
cd backend
make migrate-up
make migrate-down
make migrate-create name=add_something
```

## Frontend

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Runs on http://localhost:3000.
