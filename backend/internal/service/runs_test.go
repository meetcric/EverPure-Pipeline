package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pipelinepulse/internal/model"
)

// fakeRunStore stands in for the DB. Unset methods panic, like fakeStore.
type fakeRunStore struct {
	RunStore
	created *model.Run           // what reached Create; nil = never called
	current model.Status         // status Get reports; "" = run does not exist
	moved   bool                 // UpdateStatus succeeds when true
	runs    []model.Run          // what List returns
	query   *model.ListRunsQuery // what reached List
	exists  bool                 // what PipelineExists returns
}

func (f *fakeRunStore) Create(_ context.Context, r model.Run) (model.Run, error) {
	f.created = &r
	return r, nil
}

func (f *fakeRunStore) Get(_ context.Context, id string) (model.Run, error) {
	if f.current == "" {
		return model.Run{}, model.ErrNotFound
	}
	return model.Run{ID: id, Status: f.current}, nil
}

func (f *fakeRunStore) UpdateStatus(_ context.Context, id string, to model.Status, _ []model.Status) (model.Run, error) {
	if !f.moved {
		return model.Run{}, model.ErrNotFound
	}
	return model.Run{ID: id, Status: to}, nil
}

func (f *fakeRunStore) List(_ context.Context, q model.ListRunsQuery) ([]model.Run, error) {
	f.query = &q
	return f.runs, nil
}

func (f *fakeRunStore) PipelineExists(context.Context, string) (bool, error) {
	return f.exists, nil
}

// fakePublisher records what the service published.
type fakePublisher struct {
	published []model.Status
}

func (f *fakePublisher) Publish(_ string, s model.Status) {
	f.published = append(f.published, s)
}

func TestCreateRunValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      model.CreateRunInput
		wantErr bool
		wantSha string
	}{
		{"valid, sha lowercased", model.CreateRunInput{TriggeredBy: " ci ", CommitSha: " ABCD1234 "}, false, "abcd1234"},
		{"blank triggeredBy", model.CreateRunInput{TriggeredBy: "  ", CommitSha: "abcd"}, true, ""},
		{"triggeredBy too long", model.CreateRunInput{TriggeredBy: strings.Repeat("a", 101), CommitSha: "abcd"}, true, ""},
		{"sha too short", model.CreateRunInput{TriggeredBy: "ci", CommitSha: "abc"}, true, ""},
		{"sha too long", model.CreateRunInput{TriggeredBy: "ci", CommitSha: strings.Repeat("a", 65)}, true, ""},
		{"sha not hex", model.CreateRunInput{TriggeredBy: "ci", CommitSha: "xyz123"}, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeRunStore{}
			_, err := NewRunService(store, &fakePublisher{}).Create(context.Background(), "pl-1", tc.in)

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
			if store.created.CommitSha != tc.wantSha || store.created.TriggeredBy != "ci" {
				t.Errorf("store got %+v", store.created)
			}
			if store.created.ID == "" || store.created.PipelineID != "pl-1" {
				t.Error("service must assign an ID and keep the pipeline ID")
			}
		})
	}
}

func TestUpdateRunStatus(t *testing.T) {
	tests := []struct {
		name        string
		to          model.Status
		store       fakeRunStore
		wantErr     error // nil = success
		wantPublish bool  // only a real change is sent to SSE listeners
	}{
		{"allowed move", model.StatusRunning, fakeRunStore{moved: true}, nil, true},
		{"same status is a no-op", model.StatusPassed, fakeRunStore{current: model.StatusPassed}, nil, false},
		{"terminal run can't move", model.StatusRunning, fakeRunStore{current: model.StatusPassed}, model.ErrInvalidTransition, false},
		{"missing run", model.StatusRunning, fakeRunStore{}, model.ErrNotFound, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pub := &fakePublisher{}
			run, err := NewRunService(&tc.store, pub).UpdateStatus(context.Background(), "run-1", model.UpdateRunInput{Status: tc.to})

			wantN := 0
			if tc.wantPublish {
				wantN = 1
			}
			if len(pub.published) != wantN {
				t.Errorf("published %v, want %d event(s)", pub.published, wantN)
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if run.Status != tc.to {
				t.Errorf("status = %q, want %q", run.Status, tc.to)
			}
		})
	}

	t.Run("unknown status", func(t *testing.T) {
		_, err := NewRunService(&fakeRunStore{}, &fakePublisher{}).UpdateStatus(context.Background(), "run-1", model.UpdateRunInput{Status: "done"})
		if !isValidationErr(err) {
			t.Fatalf("want ValidationError, got %v", err)
		}
	})
}

func TestAllowedFrom(t *testing.T) {
	// queued is never a target, and terminal statuses are never a source.
	if _, ok := allowedFrom[model.StatusQueued]; ok {
		t.Error("queued must not be a target")
	}
	for to, froms := range allowedFrom {
		for _, from := range froms {
			if from.Terminal() {
				t.Errorf("%s → %s: a terminal status must not be a source", from, to)
			}
		}
	}
}

func TestListRunsValidation(t *testing.T) {
	tests := []struct {
		name string
		in   model.ListRunsInput
	}{
		{"unknown status", model.ListRunsInput{Status: "done"}},
		{"limit too low", model.ListRunsInput{Limit: -1}},
		{"limit too high", model.ListRunsInput{Limit: 101}},
		{"cursor not base64", model.ListRunsInput{Cursor: "!!!"}},
		{"cursor without id", model.ListRunsInput{Cursor: encodeCursor(model.RunCursor{StartedAt: time.Now()})}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeRunStore{}
			_, err := NewRunService(store, &fakePublisher{}).List(context.Background(), "pl-1", tc.in)
			if !isValidationErr(err) {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if store.query != nil {
				t.Fatal("invalid input must not reach the store")
			}
		})
	}
}

func TestListRunsPaging(t *testing.T) {
	now := time.Now().UTC()
	runs := []model.Run{{ID: "r3", StartedAt: now}, {ID: "r2", StartedAt: now.Add(-time.Minute)}, {ID: "r1", StartedAt: now.Add(-2 * time.Minute)}}

	t.Run("default limit asks for one extra row", func(t *testing.T) {
		store := &fakeRunStore{runs: runs[:1]}
		page, err := NewRunService(store, &fakePublisher{}).List(context.Background(), "pl-1", model.ListRunsInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.query.Limit != defaultRunLimit+1 {
			t.Errorf("store limit = %d, want %d", store.query.Limit, defaultRunLimit+1)
		}
		if page.NextCursor != nil {
			t.Error("last page must have no next cursor")
		}
	})

	t.Run("extra row means another page", func(t *testing.T) {
		page, err := NewRunService(&fakeRunStore{runs: runs}, &fakePublisher{}).List(context.Background(), "pl-1", model.ListRunsInput{Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Items) != 2 || page.NextCursor == nil {
			t.Fatalf("want 2 items and a cursor, got %d items, cursor %v", len(page.Items), page.NextCursor)
		}
		// The cursor points at the last item shown, and the next request decodes it back.
		store := &fakeRunStore{runs: runs[2:]}
		if _, err := NewRunService(store, &fakePublisher{}).List(context.Background(), "pl-1", model.ListRunsInput{Cursor: *page.NextCursor}); err != nil {
			t.Fatalf("cursor rejected: %v", err)
		}
		if store.query.After.ID != "r2" || !store.query.After.StartedAt.Equal(runs[1].StartedAt) {
			t.Errorf("store got cursor %+v, want r2", store.query.After)
		}
	})

	t.Run("empty page for missing pipeline is 404", func(t *testing.T) {
		_, err := NewRunService(&fakeRunStore{exists: false}, &fakePublisher{}).List(context.Background(), "nope", model.ListRunsInput{})
		if !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("empty page for real pipeline is fine", func(t *testing.T) {
		_, err := NewRunService(&fakeRunStore{exists: true}, &fakePublisher{}).List(context.Background(), "pl-1", model.ListRunsInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
