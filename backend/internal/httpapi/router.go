// Package httpapi holds HTTP routing and handlers for /api/v1.
package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter wires every route. Go 1.22+ ServeMux supports "METHOD /path/{param}".
func NewRouter(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler(pool))
	return mux
}
