package httpapi

import (
	"context"
	"net/http"

	"pipelinepulse/internal/model"
	"pipelinepulse/internal/utils/httpjson"
)

type RunService interface {
	Get(ctx context.Context, id string) (model.RunWithNotes, error)
	Create(ctx context.Context, pipelineID string, in model.CreateRunInput) (model.Run, error)
	UpdateStatus(ctx context.Context, id string, in model.UpdateRunInput) (model.Run, error)
}

// runHandler serves /api/v1/runs and /api/v1/pipelines/{id}/runs.
type runHandler struct {
	svc RunService
}

// get handles GET /runs/{id}: the run plus its notes.
func (h runHandler) get(w http.ResponseWriter, r *http.Request) {
	run, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, run)
}

// create handles POST /pipelines/{id}/runs. A new run is always queued.
func (h runHandler) create(w http.ResponseWriter, r *http.Request) {
	var in model.CreateRunInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	run, err := h.svc.Create(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, run)
}

// update handles PATCH /runs/{id}: a status change.
func (h runHandler) update(w http.ResponseWriter, r *http.Request) {
	var in model.UpdateRunInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	run, err := h.svc.UpdateStatus(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, run)
}
