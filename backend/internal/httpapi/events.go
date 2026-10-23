package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pipelinepulse/internal/model"
)

// RunEvents is the live feed of run status changes.
type RunEvents interface {
	Subscribe(runID string) (<-chan model.Status, func())
	Done() <-chan struct{} // closed on server shutdown
}

// eventsHandler serves GET /runs/{id}/events as Server-Sent Events.
type eventsHandler struct {
	runs   RunService
	events RunEvents
}

// heartbeatEvery keeps idle connections alive through proxies that close quiet ones.
const heartbeatEvery = 15 * time.Second

// stream sends the run's current status, then one event per status change, until the
// client disconnects or the server shuts down. Each event is `data: {"status":"..."}`.
func (h eventsHandler) stream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Subscribe before reading the current status: a change made in between is then
	// queued on the channel instead of lost.
	updates, unsubscribe := h.events.Subscribe(id)
	defer unsubscribe() // runs on every return, so a closed tab never leaves a subscriber behind

	run, err := h.runs.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, err) // still a normal JSON response: no stream headers sent yet
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // ask nginx-style proxies not to buffer the stream
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)

	last := run.Status
	if err := writeStatus(w, rc, last); err != nil {
		return
	}

	heartbeat := time.NewTicker(heartbeatEvery)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done(): // client disconnected (tab closed, EventSource.close())
			return
		case <-h.events.Done(): // server shutting down
			return
		case s := <-updates:
			// Two quick PATCHes can publish out of order; a run never moves back a stage.
			if s.Stage() <= last.Stage() {
				continue
			}
			last = s
			if err := writeStatus(w, rc, s); err != nil {
				return
			}
		case <-heartbeat.C:
			// A comment line: EventSource ignores it, but it is real traffic on the connection.
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

// writeStatus writes one SSE event and flushes it, so it leaves now instead of sitting in a buffer.
// A write error means the client is gone.
func writeStatus(w http.ResponseWriter, rc *http.ResponseController, s model.Status) error {
	b, err := json.Marshal(struct {
		Status model.Status `json:"status"`
	}{s})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return err
	}
	return rc.Flush()
}
