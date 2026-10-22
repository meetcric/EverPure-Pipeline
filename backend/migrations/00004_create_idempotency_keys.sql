-- +goose Up
CREATE TABLE idempotency_keys (
    key           TEXT        PRIMARY KEY CHECK (length(key) BETWEEN 1 AND 255),
    request_hash  TEXT        NOT NULL,
    status_code   INT,
    response_body JSON,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
