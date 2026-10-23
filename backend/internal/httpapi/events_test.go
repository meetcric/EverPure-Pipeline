package httpapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pipelinepulse/internal/model"
)

// fakeEvents hands out one unbuffered channel, so a send returns only once the
// handler has received it, and records whether the handler unsubscribed.
type fakeEvents struct {
	updates      chan model.Status
	unsubscribed chan struct{}
	done         chan struct{}
}

func (f *fakeEvents) Subscribe(string) (<-chan model.Status, func()) {
	return f.updates, func() { close(f.unsubscribed) }
}

func (f *fakeEvents) Done() <-chan struct{} { return f.done }

type fakeRunGetter struct {
	RunService
	status model.Status
}

func (f fakeRunGetter) Get(_ context.Context, id string) (model.RunWithNotes, error) {
	if f.status == "" {
		return model.RunWithNotes{}, model.ErrNotFound
	}
	return model.RunWithNotes{Run: model.Run{ID: id, Status: f.status}}, nil
}

func TestEventsStreamAndCleanup(t *testing.T) {
	ev := &fakeEvents{updates: make(chan model.Status), unsubscribed: make(chan struct{}), done: make(chan struct{})}
	h := eventsHandler{runs: fakeRunGetter{status: model.StatusQueued}, events: ev}

	ctx, disconnect := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/v1/runs/run-1/events", nil).WithContext(ctx)
	req.SetPathValue("id", "run-1")
	rec := httptest.NewRecorder()

	finished := make(chan struct{})
	go func() { h.stream(rec, req); close(finished) }()

	ev.updates <- model.StatusRunning
	ev.updates <- model.StatusQueued // stale (out of order): must be skipped
	ev.updates <- model.StatusPassed
	disconnect() // the browser closes the tab

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after the client disconnected")
	}
	select {
	case <-ev.unsubscribed:
	default:
		t.Fatal("handler returned without unsubscribing (leak)")
	}

	want := `data: {"status":"queued"}` + "\n\n" + `data: {"status":"running"}` + "\n\n" + `data: {"status":"passed"}` + "\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body =\n%q\nwant\n%q", got, want)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestEventsMissingRun(t *testing.T) {
	ev := &fakeEvents{updates: make(chan model.Status), unsubscribed: make(chan struct{}), done: make(chan struct{})}
	h := eventsHandler{runs: fakeRunGetter{}, events: ev}
	req := httptest.NewRequest("GET", "/api/v1/runs/nope/events", nil)
	req.SetPathValue("id", "nope")
	rec := httptest.NewRecorder()

	h.stream(rec, req)

	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Errorf("got %d %s, want 404 not_found", rec.Code, rec.Body.String())
	}
	select {
	case <-ev.unsubscribed:
	default:
		t.Fatal("must unsubscribe on the 404 path too")
	}
}
