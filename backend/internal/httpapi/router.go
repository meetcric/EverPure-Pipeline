// Package httpapi holds HTTP routing and handlers for /api/v1.
package httpapi

import "net/http"

// NewRouter wires every route. Go 1.22+ ServeMux supports "METHOD /path/{param}".
// It takes interfaces, not concrete types, so httpapi never imports pgx or the service package.
func NewRouter(db pinger, pipelines PipelineService, runs RunService, notes NoteService, events RunEvents) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler(db))

	ph := pipelineHandler{svc: pipelines}
	mux.HandleFunc("GET /api/v1/pipelines", ph.list)
	mux.HandleFunc("POST /api/v1/pipelines", ph.create)
	mux.HandleFunc("GET /api/v1/pipelines/{id}", ph.get)
	mux.HandleFunc("PATCH /api/v1/pipelines/{id}", ph.update)
	mux.HandleFunc("DELETE /api/v1/pipelines/{id}", ph.delete)

	rh := runHandler{svc: runs}
	mux.HandleFunc("GET /api/v1/pipelines/{id}/runs", rh.list)
	mux.HandleFunc("POST /api/v1/pipelines/{id}/runs", rh.create)
	mux.HandleFunc("GET /api/v1/runs/{id}", rh.get)
	mux.HandleFunc("PATCH /api/v1/runs/{id}", rh.update)

	eh := eventsHandler{runs: runs, events: events}
	mux.HandleFunc("GET /api/v1/runs/{id}/events", eh.stream)

	nh := noteHandler{svc: notes}
	mux.HandleFunc("POST /api/v1/runs/{id}/notes", nh.create)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", nh.delete)
	return mux
}
