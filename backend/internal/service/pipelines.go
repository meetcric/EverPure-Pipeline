// Package service holds the business rules. It knows nothing about HTTP or SQL.
package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"pipelinepulse/internal/model"
)

// PipelineStore is what the service needs from the DB layer.
// Declared here (the consumer) so tests can pass a fake.
type PipelineStore interface {
	List(ctx context.Context) ([]model.Pipeline, error)
	Get(ctx context.Context, id string) (model.Pipeline, error)
	Create(ctx context.Context, p model.Pipeline) (model.Pipeline, error)
	Delete(ctx context.Context, id string) error
}

// PipelineService implements httpapi.PipelineService.
type PipelineService struct {
	store PipelineStore
}

func NewPipelineService(store PipelineStore) *PipelineService {
	return &PipelineService{store: store}
}

func (s *PipelineService) List(ctx context.Context) ([]model.Pipeline, error) {
	return s.store.List(ctx)
}

func (s *PipelineService) Get(ctx context.Context, id string) (model.Pipeline, error) {
	return s.store.Get(ctx, id)
}

// Create validates the input, assigns a UUID and saves it. CreatedAt is set by the DB.
func (s *PipelineService) Create(ctx context.Context, in model.CreatePipelineInput) (model.Pipeline, error) {
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil {
		return model.Pipeline{}, err
	}
	if err := validateProduct(in.Product); err != nil {
		return model.Pipeline{}, err
	}
	return s.store.Create(ctx, model.Pipeline{
		ID:      uuid.NewString(),
		Name:    name,
		Product: in.Product,
	})
}

func (s *PipelineService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

const maxNameLen = 100

// validateName expects an already-trimmed name. Counts characters, not bytes.
func validateName(name string) error {
	if name == "" {
		return &model.ValidationError{Message: "name is required"}
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return &model.ValidationError{Message: "name must be at most 100 characters"}
	}
	return nil
}

func validateProduct(p model.Product) error {
	switch p {
	case model.ProductFlashArray, model.ProductFlashBlade, model.ProductHyperscale:
		return nil
	}
	return &model.ValidationError{Message: "product must be one of flasharray, flashblade, hyperscale"}
}
