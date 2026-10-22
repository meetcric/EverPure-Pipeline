package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pipelinepulse/internal/model"
)

// fakeStore stands in for the DB. Embedding the interface satisfies it; any method
// not overridden below panics if called, so an unexpected DB call fails the test.
type fakeStore struct {
	PipelineStore
	created   *model.Pipeline // what reached the store; nil = store never called
	updated   bool
	savedHash string // if set, CreateIdempotent pretends this hash was saved earlier
}

func (f *fakeStore) Create(_ context.Context, p model.Pipeline) (model.Pipeline, error) {
	f.created = &p
	return p, nil
}

func (f *fakeStore) CreateIdempotent(_ context.Context, _, hash string, p model.Pipeline, status int) (model.IdempotencyRecord, error) {
	f.created = &p
	if f.savedHash != "" {
		hash = f.savedHash
	}
	return model.IdempotencyRecord{RequestHash: hash, StatusCode: status}, nil
}

func (f *fakeStore) Update(_ context.Context, _ string, _ model.UpdatePipelineInput) (model.Pipeline, error) {
	f.updated = true
	return model.Pipeline{}, nil
}

func isValidationErr(err error) bool {
	var v *model.ValidationError
	return errors.As(err, &v)
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name     string
		in       model.CreatePipelineInput
		wantErr  bool
		wantName string // the name the store should receive
	}{
		{"trims name", model.CreatePipelineInput{Name: "  FA Nightly  ", Product: model.ProductFlashArray}, false, "FA Nightly"},
		{"blank name", model.CreatePipelineInput{Name: "   ", Product: model.ProductFlashArray}, true, ""},
		{"name too long", model.CreatePipelineInput{Name: strings.Repeat("a", 101), Product: model.ProductFlashArray}, true, ""},
		{"unknown product", model.CreatePipelineInput{Name: "X", Product: "banana"}, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			_, err := NewPipelineService(store).Create(context.Background(), tc.in)

			if tc.wantErr {
				if !isValidationErr(err) {
					t.Fatalf("want ValidationError, got %v", err)
				}
				if store.created != nil {
					t.Fatal("invalid input must not reach the store")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store.created.Name != tc.wantName {
				t.Errorf("store got name %q, want %q", store.created.Name, tc.wantName)
			}
			if store.created.ID == "" {
				t.Error("service must assign an ID")
			}
		})
	}
}

func TestUpdateRejectsEmptyPatch(t *testing.T) {
	store := &fakeStore{}
	_, err := NewPipelineService(store).Update(context.Background(), "pl-1", model.UpdatePipelineInput{})

	if !isValidationErr(err) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if store.updated {
		t.Fatal("empty PATCH must not reach the store")
	}
}

func TestCreateIdempotentHashCheck(t *testing.T) {
	in := model.CreatePipelineInput{Name: "X", Product: model.ProductFlashArray}

	t.Run("same body replays", func(t *testing.T) {
		_, err := NewPipelineService(&fakeStore{}).CreateIdempotent(context.Background(), "k1", in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("different body is rejected", func(t *testing.T) {
		store := &fakeStore{savedHash: "hash-of-another-body"}
		_, err := NewPipelineService(store).CreateIdempotent(context.Background(), "k1", in)
		if !errors.Is(err, model.ErrIdempotencyMismatch) {
			t.Fatalf("want ErrIdempotencyMismatch, got %v", err)
		}
	})
}
