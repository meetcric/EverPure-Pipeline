-- +goose Up
CREATE TABLE pipelines (
    id         TEXT        PRIMARY KEY,
    name       TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    product    TEXT        NOT NULL CHECK (product IN ('flasharray', 'flashblade', 'hyperscale')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE IF EXISTS pipelines;
