-- +goose Up
CREATE TABLE notes (
    id         TEXT        PRIMARY KEY,
    run_id     TEXT        NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    author     TEXT        NOT NULL CHECK (length(trim(author)) > 0),
    body       TEXT        NOT NULL CHECK (length(trim(body)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX notes_run_created_idx ON notes (run_id, created_at, id);

-- +goose Down
DROP TABLE IF EXISTS notes;
