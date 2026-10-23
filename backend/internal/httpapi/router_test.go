package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pipelinepulse/internal/model"
)

// Fakes for the services. Each returns err if set; unset methods panic if called.

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type fakePipelines struct {
	PipelineService
	err error
}

func (f fakePipelines) Get(_ context.Context, id string) (model.Pipeline, error) {
	return model.Pipeline{ID: id}, f.err
}

func (f fakePipelines) Create(_ context.Context, in model.CreatePipelineInput) (model.Pipeline, error) {
	return model.Pipeline{ID: "new", Name: in.Name}, f.err
}

func (f fakePipelines) CreateIdempotent(context.Context, string, model.CreatePipelineInput) (model.IdempotencyRecord, error) {
	return model.IdempotencyRecord{StatusCode: http.StatusCreated, Body: []byte(`{"id":"saved"}`)}, f.err
}

func (f fakePipelines) Delete(context.Context, string) error { return f.err }

type fakeRuns struct {
	RunService
	err error
}

func (f fakeRuns) List(context.Context, string, model.ListRunsInput) (model.RunPage, error) {
	return model.RunPage{Items: []model.Run{}}, f.err
}

func (f fakeRuns) UpdateStatus(_ context.Context, id string, in model.UpdateRunInput) (model.Run, error) {
	return model.Run{ID: id, Status: in.Status}, f.err
}

type fakeNotes struct {
	NoteService
	err error
}

func (f fakeNotes) Create(_ context.Context, runID string, in model.CreateNoteInput) (model.Note, error) {
	return model.Note{ID: "n1", RunID: runID}, f.err
}

func (f fakeNotes) Delete(context.Context, string) error { return f.err }

func TestRoutes(t *testing.T) {
	tests := []struct {
		name     string
		router   http.Handler
		method   string
		path     string
		body     string
		header   map[string]string
		wantCode int
		wantBody string // substring the response must contain; "" = don't check
	}{
		{"health ok", NewRouter(fakePinger{}, nil, nil, nil), "GET", "/api/v1/health", "", nil, 200, `"db":"ok"`},
		{"health db down", NewRouter(fakePinger{err: errors.New("down")}, nil, nil, nil), "GET", "/api/v1/health", "", nil, 503, `"db":"down"`},

		{"get pipeline", pipelinesRouter(nil), "GET", "/api/v1/pipelines/pl-1", "", nil, 200, `"id":"pl-1"`},
		{"get missing pipeline", pipelinesRouter(model.ErrNotFound), "GET", "/api/v1/pipelines/nope", "", nil, 404, `"not_found"`},
		{"create pipeline", pipelinesRouter(nil), "POST", "/api/v1/pipelines", `{"name":"x","product":"flasharray"}`, nil, 201, `"id":"new"`},
		{"create with bad JSON", pipelinesRouter(nil), "POST", "/api/v1/pipelines", `{`, nil, 400, `"invalid_json"`},
		{"create invalid input", pipelinesRouter(&model.ValidationError{Message: "name is required"}), "POST", "/api/v1/pipelines", `{}`, nil, 400, `name is required`},
		{"idempotent create replays saved body", pipelinesRouter(nil), "POST", "/api/v1/pipelines", `{}`, map[string]string{"Idempotency-Key": "k1"}, 201, `{"id":"saved"}`},
		{"idempotency key too long", pipelinesRouter(nil), "POST", "/api/v1/pipelines", `{}`, map[string]string{"Idempotency-Key": strings.Repeat("k", 256)}, 400, `Idempotency-Key`},
		{"idempotency key reused", pipelinesRouter(model.ErrIdempotencyMismatch), "POST", "/api/v1/pipelines", `{}`, map[string]string{"Idempotency-Key": "k1"}, 422, `"idempotency_key_mismatch"`},
		{"delete pipeline", pipelinesRouter(nil), "DELETE", "/api/v1/pipelines/pl-1", "", nil, 204, ""},

		{"list runs", runsRouter(nil), "GET", "/api/v1/pipelines/pl-1/runs?status=failed&limit=5", "", nil, 200, `"items":[]`},
		{"list runs, limit not a number", runsRouter(nil), "GET", "/api/v1/pipelines/pl-1/runs?limit=abc", "", nil, 400, `limit must be a number`},
		{"patch run status", runsRouter(nil), "PATCH", "/api/v1/runs/r1", `{"status":"running"}`, nil, 200, `"status":"running"`},
		{"patch run, invalid transition", runsRouter(fmt.Errorf("%w: passed → running", model.ErrInvalidTransition)), "PATCH", "/api/v1/runs/r1", `{"status":"running"}`, nil, 409, `"invalid_transition"`},

		{"create note", notesRouter(nil), "POST", "/api/v1/runs/r1/notes", `{"author":"a","body":"b"}`, nil, 201, `"runId":"r1"`},
		{"create note on missing run", notesRouter(model.ErrNotFound), "POST", "/api/v1/runs/nope/notes", `{"author":"a","body":"b"}`, nil, 404, ""},
		{"delete note", notesRouter(nil), "DELETE", "/api/v1/notes/n1", "", nil, 204, ""},

		{"unknown route", pipelinesRouter(nil), "GET", "/api/v1/nope", "", nil, 404, ""},
		{"wrong method", pipelinesRouter(nil), "PUT", "/api/v1/pipelines/pl-1", "", nil, 405, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			for k, v := range tc.header {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			tc.router.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.wantCode, w.Body)
			}
			if !strings.Contains(w.Body.String(), tc.wantBody) {
				t.Errorf("body %s does not contain %s", w.Body, tc.wantBody)
			}
		})
	}
}

func TestInternalErrorHidesCause(t *testing.T) {
	w := httptest.NewRecorder()
	pipelinesRouter(errors.New("pq: connection refused")).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/pipelines/pl-1", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "connection refused") {
		t.Errorf("500 must not leak the cause: %s", w.Body)
	}
}

func pipelinesRouter(err error) http.Handler {
	return NewRouter(fakePinger{}, fakePipelines{err: err}, nil, nil)
}

func runsRouter(err error) http.Handler {
	return NewRouter(fakePinger{}, nil, fakeRuns{err: err}, nil)
}

func notesRouter(err error) http.Handler {
	return NewRouter(fakePinger{}, nil, nil, fakeNotes{err: err})
}
