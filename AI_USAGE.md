# AI Usage

## Tools used

Claude Code in VS Code (Claude Opus 5.5, some Claude Sonnet 5.5).

## How I worked

I treated Claude like a fast pair who types: I set the design, it wrote the code, I reviewed it piece by piece and pushed back where it was wrong or over-built. Most of the code is AI-written; the decisions and the corrections are mine.

## Where

- **Mostly AI-generated:** the Go backend, all tests, the frontend, Dockerfiles, `docker-compose.yml`, `scripts/seed.sql`, most migrations, first drafts of the docs.
- **AI-assisted:** `00002_create_runs.sql` (my draft, Claude fixed it), `00003_create_notes.sql` (Claude's, I edited it).
- **Mine:** the design. Layering, migrations as a separate step, seeding in SQL, the idempotency design (PK as lock, 422 on mismatch, hash the decoded input), status transitions enforced in one SQL statement, an in-process SSE broker, nginx proxy instead of CORS, the frontend's api/state/ui split. Plus small fixes like the `db.go` leak and removing the Postgres host port.

## Prompts

1. *"Implement GET /runs/{id}, POST /pipelines/{id}/runs, PATCH /runs/{id}. Check the doc for more details."*
2. *"Can we not directly seed data into db without creating a service?"*
3. *"Implement this changes"*, after we agreed on the idempotency design.
4. *"Keep API access, state, and presentation separable ... Have this separation."* This was a review of Claude's first frontend. It led to the `api/` → `state/` → `ui/` split, moving idempotency keys and retry policy out of the components, and lint rules that fail the build if a page imports the API layer.
5. *"Write the testcase ... only write the required one, don't write extensive test case."*

## What I changed

1. **Caught a resource leak.** Claude's DB retry loop used `defer cancel()` inside the loop, so every failed ping kept its timer alive until the function returned. I moved it to `cancel()` right after each ping.
2. **Cut an over-built seeder.** Claude wrote a Go program plus an extra compose service to load one JSON file. I replaced it with one SQL file: Postgres parses the JSON itself (`jsonb_to_recordset`) in a single transaction, and it's safe to re-run.
3. **Moved migrations out of the API.** Claude ran them on API startup. I put them in a one-shot container that the API waits for. A bad migration then blocks the deploy instead of crash-looping the API, and it runs once no matter how many replicas there are.
4. **Changed ids to fit the provided seed.** I first chose UUID columns. Claude pointed out that the seed uses ids like `pl-flasharray-nightly`, so I switched to `TEXT` ids: new rows still get UUIDs, and the seed file stays unmodified as the brief requires.
5. **Made the frontend smaller and enforced its layers.** The first version was 29 files. I had it cut to about 10, and asked for lint rules so `ui/` can't import `api/`. I broke the rule on purpose to check that lint catches it.

## Verification

I didn't trust the unit tests alone, since they use fakes. So I tested the real stack:
- **Concurrency:** 20 parallel POSTs with one Idempotency-Key create exactly one row; 12 parallel "finish" PATCHes leave exactly one winner.
- **End to end:** a 154-check script covering every endpoint, the full 5×5 status transition matrix, paging with tied timestamps, and data surviving `restart` and `down && up`.
- **Query plan:** `EXPLAIN` on the runs list shows an index-only scan with no sort.
- **SSE:** checked with curl. Live updates arrive, a no-op PATCH sends nothing, the stream closes on disconnect, and shutdown with streams open takes about 1s.
- **Static checks and tests:** `go test -race`, `go vet`, strict `tsc`, `oxlint`, Vitest (including 50 mount/unmount cycles leaving no open EventSource).
- **Browser memory check:** a headless-Chrome script (not in the repo), run by a Claude subagent, went `/` → `/runs/:id` → `/` 50 times, and 200 times as a stress run. Back on `/` each time there were 0 live intervals and 0 open EventSources, with DOM nodes and listeners flat. The heap grew by about 0.5 MB and levelled off; a heap-snapshot diff put that down to V8 compiled code and DevTools buffers, not app objects.

## Assistant Attestation

- Models: Claude Opus 5.5 (`claude-opus-5-5`), Claude Sonnet 5.5 (`claude-sonnet-5-5`).
- Files generated or substantially edited: `README.md`, `AI_USAGE.md`, `NOTES.md`, `docker-compose.yml`, `Makefile`, `scripts/seed.sql`, `backend/Dockerfile`, `backend/cmd/*`, `backend/migrations/*`, `backend/internal/*` (including tests), and `frontend/` apart from the `npm create vite` scaffold.
- Dates: 2026-10-20 to 2026-10-26.
