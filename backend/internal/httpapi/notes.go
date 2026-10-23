package httpapi

import (
	"context"
	"net/http"

	"pipelinepulse/internal/model"
	"pipelinepulse/internal/utils/httpjson"
)

type NoteService interface {
	Create(ctx context.Context, runID string, in model.CreateNoteInput) (model.Note, error)
	Delete(ctx context.Context, id string) error
}

// noteHandler serves /api/v1/runs/{id}/notes and /api/v1/notes/{id}.
type noteHandler struct {
	svc NoteService
}

// create handles POST /runs/{id}/notes.
func (h noteHandler) create(w http.ResponseWriter, r *http.Request) {
	var in model.CreateNoteInput
	if err := httpjson.Decode(w, r, &in); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	note, err := h.svc.Create(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, note)
}

// delete handles DELETE /notes/{id}. 204 has no body.
func (h noteHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
