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
	CreateIdempotent(ctx context.Context, key string, in model.CreatePipelineInput) (model.IdempotencyRecord, error)
	Update(ctx context.Context, id string, in model.UpdatePipelineInput) (model.Pipeline, error)
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

// The client generates the key once per action and resends it on retries. It is optional.
const (
	idempotencyHeader    = "Idempotency-Key"
	maxIdempotencyKeyLen = 255
)

// create handles POST /pipelines. With an Idempotency-Key, retries replay the first response.
func (h pipelineHandler) create(w http.ResponseWriter, r *http.Request) {
	var in model.CreatePipelineInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if key := r.Header.Get(idempotencyHeader); key != "" {
		if len(key) > maxIdempotencyKeyLen {
			httpjson.Error(w, http.StatusBadRequest, "validation_error", "Idempotency-Key must be at most 255 characters")
			return
		}
		rec, err := h.svc.CreateIdempotent(r.Context(), key, in)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		httpjson.WriteRaw(w, rec.StatusCode, rec.Body)
		return
	}

	p, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, p)
}

func (h pipelineHandler) update(w http.ResponseWriter, r *http.Request) {
	var in model.UpdatePipelineInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	p, err := h.svc.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, p)
}

// delete handles DELETE /pipelines/{id}. 204 has no body.
func (h pipelineHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
