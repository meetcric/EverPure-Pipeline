package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"pipelinepulse/internal/model"
)

// RunStore implements service.RunStore.
type RunStore struct {
	pool *pgxpool.Pool
}

func NewRunStore(pool *pgxpool.Pool) *RunStore {
	return &RunStore{pool: pool}
}

const runColumns = `id, pipeline_id, status, started_at, finished_at, duration_ms, triggered_by, commit_sha`

func (s *RunStore) Get(ctx context.Context, id string) (model.Run, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+runColumns+` FROM runs WHERE id = $1`, id)
	if err != nil {
		return model.Run{}, fmt.Errorf("get run: %w", err)
	}
	return collectRun(rows, "get run")
}

// fkViolation is Postgres's error code for a foreign key that points at a missing row.
const fkViolation = "23503"

// Create inserts a run. status ('queued') and started_at (now()) come from the column defaults.
func (s *RunStore) Create(ctx context.Context, r model.Run) (model.Run, error) {
	rows, err := s.pool.Query(ctx,
		`INSERT INTO runs (id, pipeline_id, triggered_by, commit_sha) VALUES ($1, $2, $3, $4)
		 RETURNING `+runColumns,
		r.ID, r.PipelineID, r.TriggeredBy, r.CommitSha)
	if err != nil {
		return model.Run{}, fmt.Errorf("create run: %w", err)
	}
	run, err := collectRun(rows, "create run")
	// The pipeline doesn't exist: the FK rejected the row, so no separate lookup is needed.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == fkViolation {
		return model.Run{}, model.ErrNotFound
	}
	return run, err
}

// UpdateStatus sets the status only if the current one is in from (compare-and-set).
// A terminal status also sets finished_at and duration_ms. If no row matches
// (missing run or disallowed move) it returns ErrNotFound and the caller works out which.
func (s *RunStore) UpdateStatus(ctx context.Context, id string, to model.Status, from []model.Status) (model.Run, error) {
	fromStr := make([]string, len(from))
	for i, st := range from {
		fromStr[i] = string(st)
	}
	rows, err := s.pool.Query(ctx,
		`UPDATE runs SET
		   status      = $2,
		   finished_at = CASE WHEN $4 THEN now() END,
		   duration_ms = CASE WHEN $4 THEN (EXTRACT(EPOCH FROM now() - started_at) * 1000)::bigint END
		 WHERE id = $1 AND status = ANY($3)
		 RETURNING `+runColumns,
		id, string(to), fromStr, to.Terminal())
	if err != nil {
		return model.Run{}, fmt.Errorf("update run status: %w", err)
	}
	return collectRun(rows, "update run status")
}

// ListNotes returns the run's notes, oldest first ([] when there are none).
func (s *RunStore) ListNotes(ctx context.Context, runID string) ([]model.Note, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, run_id, author, body, created_at FROM notes WHERE run_id = $1 ORDER BY created_at, id`, runID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	notes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Note, error) {
		var n model.Note
		err := row.Scan(&n.ID, &n.RunID, &n.Author, &n.Body, &n.CreatedAt)
		return n, err
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	return notes, nil
}

func scanRun(row pgx.CollectableRow) (model.Run, error) {
	var r model.Run
	err := row.Scan(&r.ID, &r.PipelineID, &r.Status, &r.StartedAt, &r.FinishedAt, &r.DurationMs, &r.TriggeredBy, &r.CommitSha)
	return r, err
}

func collectRun(rows pgx.Rows, op string) (model.Run, error) {
	r, err := pgx.CollectOneRow(rows, scanRun)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Run{}, model.ErrNotFound
	}
	if err != nil {
		return model.Run{}, fmt.Errorf("%s: %w", op, err)
	}
	return r, nil
}
