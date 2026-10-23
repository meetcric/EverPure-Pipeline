package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"pipelinepulse/internal/model"
)

// NoteStore is what the note service needs from the DB layer.
type NoteStore interface {
	Create(ctx context.Context, n model.Note) (model.Note, error)
	Delete(ctx context.Context, id string) error
}

// NoteService implements httpapi.NoteService.
type NoteService struct {
	store NoteStore
}

func NewNoteService(store NoteStore) *NoteService {
	return &NoteService{store: store}
}

const maxNoteBodyLen = 5000

// Create validates the input and saves the note. The store returns ErrNotFound
// if the run does not exist.
func (s *NoteService) Create(ctx context.Context, runID string, in model.CreateNoteInput) (model.Note, error) {
	author := strings.TrimSpace(in.Author)
	if author == "" {
		return model.Note{}, &model.ValidationError{Message: "author is required"}
	}
	if utf8.RuneCountInString(author) > maxNameLen {
		return model.Note{}, &model.ValidationError{Message: "author must be at most 100 characters"}
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return model.Note{}, &model.ValidationError{Message: "body is required"}
	}
	if utf8.RuneCountInString(body) > maxNoteBodyLen {
		return model.Note{}, &model.ValidationError{Message: "body must be at most 5000 characters"}
	}
	return s.store.Create(ctx, model.Note{ID: uuid.NewString(), RunID: runID, Author: author, Body: body})
}

func (s *NoteService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
