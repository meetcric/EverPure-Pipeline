# AI Usage

## Tools used

- Claude Code (VS Code extension)

## Where

*Filled in as the project progresses.*

## Prompts

1. Project structure: *"Follow the structure of the code given in the PRD, put the code inside Code folder."*
2. Migration runner: *"Make the changes"*, i.e. write `cmd/migrate/main.go` using goose (with embedded SQL files), after we had agreed on the design: a separate migrate container that the api waits for.

## What I changed

1. **Seeding: replaced a Go seeder and an extra compose service with one SQL file.** Claude first implemented `make seed` as a separate Go program (`cmd/seed`) plus a dedicated `seed` service in `docker-compose.yml`. I didn't want a whole service and two Go files just to load a JSON file. I asked for alternatives, and chose to have Postgres parse the JSON itself: `make seed` pipes `scripts/seed.sql` into `psql` and passes `seed.json` in as a variable, and `jsonb_to_recordset` turns it into rows. The result is less code, no extra service, and psql reports the rows actually inserted (`INSERT 0 3`, then `INSERT 0 0` on re-run). The trade-off I accepted is that the stack must be running before `make seed`.

## Verification

*TBD*

## Assistant Attestation

- Model: Claude Opus 5.5 (`claude-opus-5-5`)
- Date: 2026-10-20