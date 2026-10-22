// Package httpjson holds the JSON helpers every handler uses, so the response
// and error shapes are defined in one place.
package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// Write sends v as JSON with the given status code.
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent, so if encoding fails all we can do is log it.
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// errorBody is the shape the brief requires: {"error": {"code": "...", "message": "..."}}.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error sends an error response in the standard shape.
func Error(w http.ResponseWriter, status int, code, message string) {
	Write(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// maxBodyBytes caps request bodies. A pipeline is a few hundred bytes, so 1 MB is generous.
const maxBodyBytes = 1 << 20

// Decode reads exactly one JSON object from the body into dst.
// It rejects unknown fields, empty bodies, bodies over 1 MB and trailing data.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return err
	}
	// Anything after the first object (e.g. `{}{}`) is a malformed request.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must be a single JSON object")
	}
	return nil
}
