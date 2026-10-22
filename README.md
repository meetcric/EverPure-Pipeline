# Pipeline Pulse

Assignment ID: **EVP-FS-TH-01**

A dashboard for viewing and managing CI pipeline runs: a Go REST API with Postgres, plus a React + TypeScript frontend.

## Run

Requires Docker with Compose v2. No `.env` or other setup is needed.

```bash
docker compose up --build        # add -d to run in the background
```

This starts, in order:
1. `db`: Postgres 16, with data stored in the `pgdata` volume
2. `migrate`: applies the SQL migrations in `backend/migrations/`, then exits
3. `api`: the REST API on port 8080, started only after migrations succeed

Check that it's up:

```bash
curl http://localhost:8080/api/v1/health   # {"db":"ok","status":"ok"}
```

- API: http://localhost:8080/api/v1
- Frontend: http://localhost:3000 (coming soon)

## Seed data

`docker compose up` starts with an empty database. To load `seed/seed.json`, run this **after** the stack is up:

```bash
make seed
```

- It pipes [`scripts/seed.sql`](scripts/seed.sql) into `psql` in the `db` container, passing `seed.json` in as a variable. Postgres's `jsonb_to_recordset` turns the JSON arrays into rows.
- It's safe to run more than once. Existing rows are skipped (`ON CONFLICT DO NOTHING`), so nothing is duplicated.
- It's all one transaction, so either the whole seed is loaded or nothing is.
- Seeding never runs automatically on `up`. Otherwise, data deleted through the API would come back after a restart.

Other useful commands:

```bash
docker compose logs -f api                         # follow API logs
docker compose exec db psql -U user -d pulse       # open a SQL shell
docker compose down                                # stop (data is kept)
docker compose down -v                             # stop and delete all data
```

## Repository layout

```
backend/     Go API
  cmd/api      REST API server
  cmd/migrate  applies SQL migrations (runs before the API)
  migrations/  goose SQL migrations
scripts/     seed.sql (used by `make seed`)
Makefile     `make seed`
frontend/    React + TypeScript app
seed/        seed.json (provided, unmodified)
```

## Decisions & trade-offs

_Written as the project progresses._

## With more time

_TBD_
