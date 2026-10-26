# Notes

## Known gaps (and how I'd fix them)

- **SSE won't scale past one API container.** The broker is in-process, so a PATCH on container A never reaches a stream on container B. Fix: `NOTIFY run_status` after the update and `LISTEN` in each container. Postgres is already there, so no Redis is needed.
- **Idempotency keys never expire.** Add a `created_at` TTL (e.g. 24h) and a periodic delete.
- **The SQL has no automated tests.** The tricky parts (the idempotency transaction, the compare-and-set status update, the cursor query) were verified by hand and with a script. Next step: integration tests against a throwaway Postgres.
- **`durationMs` includes queue time,** because `startedAt` is set at creation, matching how the seed data uses it. A separate `queuedAt` would give true run time and a queue-wait metric.
- **Unknown routes return plain-text 404/405.** A small wrapper around the mux would return the standard JSON error.
- **Only the run detail page is live.** The runs list could subscribe to one per-pipeline stream rather than one per run.

## Ideas for Pipeline Pulse

- **Flaky detection from data, not by hand:** flag tests whose runs flip between pass and fail on the same commit.
- **Failure grouping:** cluster failed runs by error signature, so "iSCSI timeout on node 3" shows up as one issue across 20 runs instead of 20 notes.
- **Trends per pipeline:** pass rate and p50/p95 duration over time, computed in SQL. That's enough to spot a slowing soak test early.
- **Auth:** take the author from the login instead of free text, and scope pipelines to teams.
