# Pipeline Pulse

Assignment ID: **EVP-FS-TH-01**

A dashboard for viewing and managing CI pipeline runs: a Go REST API with Postgres, plus a React + TypeScript frontend.

## Run

```bash
docker compose up --build
```

- API: http://localhost:8080/api/v1/health
- Frontend: http://localhost:3000 (coming soon)

## Repository layout

```
backend/     Go API (cmd/, internal/, migrations/)
frontend/    React + TypeScript app
seed/        seed.json (provided, unmodified)
```

## Decisions & trade-offs

_Written as the project progresses._

## With more time

_TBD_
