# DPDP

## Structure

- `frontend/` — Next.js + shadcn/ui
- `backend/` — Go + Gin API

## Getting started

```bash
cp .env.example .env
docker compose up --build
```

This starts Postgres (with pgvector), Redis, Mailpit, runs migrations, and starts the backend API on `APP_PORT` (default `8080`). Mailpit UI is available on `MAILPIT_UI_PORT` (default `8025`).

Run the frontend separately:

```bash
cd frontend
npm install
npm run dev
```

## Backend migrations

```bash
cd backend
make migrate-up
make migrate-down
make migrate-create name=add_something
```
