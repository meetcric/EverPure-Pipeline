package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"pipelinepulse/internal/model"
	"pipelinepulse/internal/utils/httpjson"
)

// writeServiceError turns an error from the service layer into an HTTP response.
// This is the only place that decides 400 vs 404 vs 500.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var vErr *model.ValidationError
	switch {
	case errors.As(err, &vErr):
		httpjson.Error(w, http.StatusBadRequest, "validation_error", vErr.Message)
	case errors.Is(err, model.ErrNotFound):
		httpjson.Error(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, model.ErrIdempotencyMismatch):
		httpjson.Error(w, http.StatusUnprocessableEntity, "idempotency_key_mismatch", err.Error())
	default:
		// Log the real cause, but never send DB or internal details to the client.
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		httpjson.Error(w, http.StatusInternalServerError, "internal", "internal error")
	}
}
