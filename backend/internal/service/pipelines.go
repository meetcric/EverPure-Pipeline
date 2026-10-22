// Package service holds the business rules. It knows nothing about SQL. The only HTTP
// detail is the status code saved with an idempotent response, so replays return it too.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
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
	CreateIdempotent(ctx context.Context, key, requestHash string, p model.Pipeline, status int) (model.IdempotencyRecord, error)
	Update(ctx context.Context, id string, in model.UpdatePipelineInput) (model.Pipeline, error)
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
	p, err := newPipeline(in)
	if err != nil {
		return model.Pipeline{}, err
	}
	return s.store.Create(ctx, p)
}

func (s *PipelineService) CreateIdempotent(ctx context.Context, key string, in model.CreatePipelineInput) (model.IdempotencyRecord, error) {
	p, err := newPipeline(in)
	if err != nil {
		return model.IdempotencyRecord{}, err
	}
	// Hash the cleaned input, not the raw bytes: same meaning → same hash,
	// whatever the whitespace or key order.
	hash, err := hashInput(model.CreatePipelineInput{Name: p.Name, Product: p.Product})
	if err != nil {
		return model.IdempotencyRecord{}, err
	}
	rec, err := s.store.CreateIdempotent(ctx, key, hash, p, http.StatusCreated)
	if err != nil {
		return model.IdempotencyRecord{}, err
	}
	if rec.RequestHash != hash {
		return model.IdempotencyRecord{}, model.ErrIdempotencyMismatch
	}
	return rec, nil
}

// newPipeline trims and validates the input and assigns a new UUID.
func newPipeline(in model.CreatePipelineInput) (model.Pipeline, error) {
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil {
		return model.Pipeline{}, err
	}
	if err := validateProduct(in.Product); err != nil {
		return model.Pipeline{}, err
	}
	return model.Pipeline{ID: uuid.NewString(), Name: name, Product: in.Product}, nil
}

// hashInput returns the SHA-256 of the input as JSON. The struct fixes the field order.
func hashInput(in model.CreatePipelineInput) (string, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (s *PipelineService) Update(ctx context.Context, id string, in model.UpdatePipelineInput) (model.Pipeline, error) {
	if in.Name == nil && in.Product == nil {
		return model.Pipeline{}, &model.ValidationError{Message: "nothing to update: send name and/or product"}
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if err := validateName(name); err != nil {
			return model.Pipeline{}, err
		}
		in.Name = &name
	}
	if in.Product != nil {
		if err := validateProduct(*in.Product); err != nil {
			return model.Pipeline{}, err
		}
	}
	return s.store.Update(ctx, id, in)
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
