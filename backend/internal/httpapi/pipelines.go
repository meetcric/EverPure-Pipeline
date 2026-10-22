package httpapi

import (
	"context"
	"net/http"

	"pipelinepulse/internal/model"
	"pipelinepulse/internal/utils/httpjson"
)

type PipelineService interface {
	List(ctx context.Context) ([]model.Pipeline, error)
	Get(ctx context.Context, id string) (model.Pipeline, error)
	Create(ctx context.Context, in model.CreatePipelineInput) (model.Pipeline, error)
	Delete(ctx context.Context, id string) error
}

// pipelineHandler serves /api/v1/pipelines. Each method: read request → call service → respond.
type pipelineHandler struct {
	svc PipelineService
}

// list handles GET /pipelines.
func (h pipelineHandler) list(w http.ResponseWriter, r *http.Request) {
	pipelines, err := h.svc.List(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, pipelines)
}

// get handles GET /pipelines/{id}.
func (h pipelineHandler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, p)
}

// create handles POST /pipelines.
func (h pipelineHandler) create(w http.ResponseWriter, r *http.Request) {
	var in model.CreatePipelineInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	p, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, p)
}

// delete handles DELETE /pipelines/{id}. 204 has no body.
func (h pipelineHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
