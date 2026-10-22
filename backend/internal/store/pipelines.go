// Package store is the DB layer: raw SQL through pgxpool. It is the only package that speaks SQL.
package store

import (
	"context"
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

func (s *PipelineStore) Create(ctx context.Context, p model.Pipeline) (model.Pipeline, error) {
	rows, err := s.pool.Query(ctx,
		`INSERT INTO pipelines (id, name, product) VALUES ($1, $2, $3)
		 RETURNING id, name, product, created_at`,
		p.ID, p.Name, p.Product)
	if err != nil {
		return model.Pipeline{}, fmt.Errorf("create pipeline: %w", err)
	}
	return collectOne(rows, "create pipeline")
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
