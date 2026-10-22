// Package store is the DB layer: raw SQL through pgxpool. It is the only package that speaks SQL.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pipelinepulse/internal/model"
)

// PipelineStore implements service.PipelineStore.
type PipelineStore struct {
	pool *pgxpool.Pool
}

func NewPipelineStore(pool *pgxpool.Pool) *PipelineStore {
	return &PipelineStore{pool: pool}
}

func (s *PipelineStore) List(ctx context.Context) ([]model.Pipeline, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, product, created_at FROM pipelines ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	pipelines, err := pgx.CollectRows(rows, scanPipeline)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	return pipelines, nil
}

func (s *PipelineStore) Get(ctx context.Context, id string) (model.Pipeline, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, product, created_at FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return model.Pipeline{}, fmt.Errorf("get pipeline: %w", err)
	}
	return collectOne(rows, "get pipeline")
}

const insertPipelineSQL = `INSERT INTO pipelines (id, name, product) VALUES ($1, $2, $3)
	RETURNING id, name, product, created_at`

func (s *PipelineStore) Create(ctx context.Context, p model.Pipeline) (model.Pipeline, error) {
	rows, err := s.pool.Query(ctx, insertPipelineSQL, p.ID, p.Name, p.Product)
	if err != nil {
		return model.Pipeline{}, fmt.Errorf("create pipeline: %w", err)
	}
	return collectOne(rows, "create pipeline")
}

func (s *PipelineStore) CreateIdempotent(ctx context.Context, key, requestHash string, p model.Pipeline, status int) (model.IdempotencyRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("begin: %w", err)
	}
	// No-op after Commit; on any early return it releases the claimed key.
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`INSERT INTO idempotency_keys (key, request_hash) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`,
		key, requestHash)
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("claim idempotency key: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Key already used and committed: return the saved response.
		var rec model.IdempotencyRecord
		err := tx.QueryRow(ctx,
			`SELECT request_hash, status_code, response_body FROM idempotency_keys WHERE key = $1`, key,
		).Scan(&rec.RequestHash, &rec.StatusCode, &rec.Body)
		if err != nil {
			return model.IdempotencyRecord{}, fmt.Errorf("read idempotency key: %w", err)
		}
		return rec, tx.Commit(ctx)
	}

	// We own the key: create the pipeline and save the response in the same transaction.
	rows, err := tx.Query(ctx, insertPipelineSQL, p.ID, p.Name, p.Product)
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("create pipeline: %w", err)
	}
	created, err := collectOne(rows, "create pipeline")
	if err != nil {
		return model.IdempotencyRecord{}, err
	}
	body, err := json.Marshal(created)
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("encode response: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE idempotency_keys SET status_code = $2, response_body = $3 WHERE key = $1`,
		key, status, body); err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("save idempotency response: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("commit: %w", err)
	}
	return model.IdempotencyRecord{RequestHash: requestHash, StatusCode: status, Body: body}, nil
}

// Update sets only the sent fields: a nil pointer becomes NULL, and COALESCE(NULL, name) keeps the old value.
func (s *PipelineStore) Update(ctx context.Context, id string, in model.UpdatePipelineInput) (model.Pipeline, error) {
	rows, err := s.pool.Query(ctx,
		`UPDATE pipelines SET name = COALESCE($2, name), product = COALESCE($3, product)
		 WHERE id = $1
		 RETURNING id, name, product, created_at`,
		id, in.Name, in.Product)
	if err != nil {
		return model.Pipeline{}, fmt.Errorf("update pipeline: %w", err)
	}
	return collectOne(rows, "update pipeline")
}

func (s *PipelineStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanPipeline(row pgx.CollectableRow) (model.Pipeline, error) {
	var p model.Pipeline
	err := row.Scan(&p.ID, &p.Name, &p.Product, &p.CreatedAt)
	return p, err
}

func collectOne(rows pgx.Rows, op string) (model.Pipeline, error) {
	p, err := pgx.CollectOneRow(rows, scanPipeline)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Pipeline{}, model.ErrNotFound
	}
	if err != nil {
		return model.Pipeline{}, fmt.Errorf("%s: %w", op, err)
	}
	return p, nil
}
