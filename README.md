# Pipeline Pulse

Assignment ID: **EVP-FS-TH-01**

Dashboard for CI pipeline runs. Go API + Postgres, React + TypeScript frontend.

## Run

```bash
docker compose up --build
make seed                      # after the stack is up; safe to run twice
```

- App: http://localhost:3000
- API: http://localhost:8080/api/v1 (`/health` → `{"status":"ok","db":"ok"}`)

Order: `db` → `migrate` (runs once, exits) → `api` → `frontend` (nginx, proxies `/api`).

Tests:

```bash
cd backend && go test -race ./...
cd frontend && npm test
```

## Layout

```
backend/
  cmd/api, cmd/migrate
  migrations/            goose SQL files
  internal/
    httpapi/  → service/  → store/      handlers → rules → SQL
    model/                              shared types
    events/                             SSE pub/sub
frontend/src/
  api/  → state/  → ui/                 fetch → TanStack Query hooks → pages
scripts/seed.sql                        used by `make seed`
```

## Decisions & trade-offs

- **Packages by layer.** Imports only go down, so handlers never touch SQL. Fine for 3 entities.
- **Run status rules in one SQL `UPDATE ... WHERE status = ANY(...)`**, so two concurrent PATCHes can't both win. Illegal move → 409.
- **Idempotency in one transaction**, with the key's primary key as the lock. Same key + same body replays the response; different body → 422.
- **Cursor pagination** (`startedAt|id`) instead of OFFSET, so pages don't shift when new runs arrive.
- **SSE via an in-process broker.** Simple, but only works with one API container.
- **Cascade delete in the DB** (`ON DELETE CASCADE`).
- **Migrations in a separate container**; the API won't start on a broken schema.
- **nginx proxy instead of CORS**: one origin, no CORS code.
- **Frontend cleanup**: requests are cancelled, the EventSource is closed and timers are cleared when you leave a page.
- **`startedAt` is the creation time**, so `durationMs` includes queue time.

## With more time

- Playwright tests, including the 50-navigation memory check.
- Postgres `LISTEN/NOTIFY` so SSE works across several API containers.
- Expire old idempotency keys.
