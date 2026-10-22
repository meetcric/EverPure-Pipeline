package httpapi

import (
	"context"
	"net/http"
	"time"

	"pipelinepulse/internal/utils/httpjson"
)

// pinger is the only thing health needs from the DB, which keeps it easy to fake in tests.
type pinger interface {
	Ping(ctx context.Context) error
}

// healthHandler really talks to the DB: 200 if a ping succeeds, 503 otherwise.
func healthHandler(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status, body := http.StatusOK, map[string]string{"status": "ok", "db": "ok"}
		if err := db.Ping(ctx); err != nil {
			status, body = http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": "down"}
		}
		httpjson.Write(w, status, body)
	}
}
