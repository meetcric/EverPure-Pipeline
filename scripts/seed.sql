-- Loads seed/seed.json into the database. Run via `make seed`, which passes
-- the JSON file in as the psql variable :data.
--
-- One transaction: either everything is inserted or nothing is.
-- ON CONFLICT DO NOTHING: re-running skips rows that already exist.
-- jsonb_to_recordset turns a JSON array into rows. Quoted names ("pipelineId")
-- must match the JSON keys exactly, because SQL identifiers are lower-cased otherwise.

BEGIN;

-- Parents first, so foreign keys are satisfied.
INSERT INTO pipelines (id, name, product)
SELECT id, name, product
FROM jsonb_to_recordset(:'data'::jsonb -> 'pipelines')
     AS p(id text, name text, product text)
ON CONFLICT (id) DO NOTHING;

INSERT INTO runs (id, pipeline_id, status, started_at, finished_at, duration_ms, triggered_by, commit_sha)
SELECT id, "pipelineId", status, "startedAt", "finishedAt", "durationMs", "triggeredBy", "commitSha"
FROM jsonb_to_recordset(:'data'::jsonb -> 'runs')
     AS r(id text, "pipelineId" text, status text, "startedAt" timestamptz,
          "finishedAt" timestamptz, "durationMs" bigint, "triggeredBy" text, "commitSha" text)
ON CONFLICT (id) DO NOTHING;

INSERT INTO notes (id, run_id, author, body)
SELECT id, "runId", author, body
FROM jsonb_to_recordset(:'data'::jsonb -> 'notes')
     AS n(id text, "runId" text, author text, body text)
ON CONFLICT (id) DO NOTHING;

COMMIT;
