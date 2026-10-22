-- +goose Up
CREATE TABLE runs (
    id           TEXT        PRIMARY KEY,
    pipeline_id  TEXT        NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    status       TEXT        NOT NULL DEFAULT 'queued'
                             CHECK (status IN ('queued', 'running', 'passed', 'failed', 'flaky')),
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ CHECK (finished_at >= started_at),
    duration_ms  BIGINT      CHECK (duration_ms >= 0),
    triggered_by TEXT        NOT NULL,
    commit_sha   TEXT        NOT NULL
);

CREATE INDEX runs_pipeline_started_idx ON runs (pipeline_id, started_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS runs;
