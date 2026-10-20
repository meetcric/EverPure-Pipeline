// Command api runs the Pipeline Pulse REST API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pipelinepulse/internal/db"
	"pipelinepulse/internal/httpapi"
)

func main() {
	// Main() stays tinny and all logic inside run() so that
	// we can use defer for cleanup and graceful shutdown.
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1) // it skips defer cleanups
	}
}

func run() error {
	// ctx is cancelled on Ctrl+C or `docker stop` (SIGTERM), which triggers graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Connect to the database.
	pool, err := db.Connect(ctx, envOr("DATABASE_URL", "postgres://user:password@localhost:5432/pulse?sslmode=disable"))
	if err != nil {
		return err
	}
	defer pool.Close()

	srv := &http.Server{
		Addr:    ":" + envOr("PORT", "8080"),
		Handler: httpapi.NewRouter(pool),
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
