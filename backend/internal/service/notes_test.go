package service

import (
	"context"
	"strings"
	"testing"

	"pipelinepulse/internal/model"
)

type fakeNoteStore struct {
	NoteStore
	created *model.Note // nil = store never called
}

func (f *fakeNoteStore) Create(_ context.Context, n model.Note) (model.Note, error) {
	f.created = &n
	return n, nil
}

func TestCreateNoteValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      model.CreateNoteInput
		wantErr bool
	}{
		{"trims author and body", model.CreateNoteInput{Author: "  meet  ", Body: "  looks flaky  "}, false},
		{"blank author", model.CreateNoteInput{Author: " ", Body: "x"}, true},
		{"author too long", model.CreateNoteInput{Author: strings.Repeat("a", 101), Body: "x"}, true},
		{"blank body", model.CreateNoteInput{Author: "meet", Body: " "}, true},
		{"body too long", model.CreateNoteInput{Author: "meet", Body: strings.Repeat("a", 5001)}, true},
		{"body at the limit, multibyte", model.CreateNoteInput{Author: "meet", Body: strings.Repeat("é", 5000)}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeNoteStore{}
			_, err := NewNoteService(store).Create(context.Background(), "run-1", tc.in)

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
			if store.created.ID == "" || store.created.RunID != "run-1" {
				t.Errorf("store got %+v, want an ID and run-1", store.created)
			}
			if store.created.Author != strings.TrimSpace(tc.in.Author) || store.created.Body != strings.TrimSpace(tc.in.Body) {
				t.Errorf("store got untrimmed values: %+v", store.created)
			}
		})
	}
}
