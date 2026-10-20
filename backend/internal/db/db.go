package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxpool.New(ctx, url) builds a connection pool using the pgx driver.
//
//	A pool keeps several open connections and reuses them
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	const attempts = 30
	for i := 1; i <= attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = pool.Ping(pingCtx)
		defer cancel()
		if err == nil {
			return pool, nil
		}
		slog.Warn("db not ready, retrying", "attempt", i, "err", err)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("db unreachable after %d attempts: %w", attempts, err)
}
