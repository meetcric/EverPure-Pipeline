package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql
	"github.com/pressly/goose/v3"

	"pipelinepulse/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	// goose works with database/sql, not pgxpool, so open a plain *sql.DB.
	db, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	// Applies only versions not yet recorded in goose_db_version, so re-runs are no-ops.
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	slog.Info("migrations up to date")
	return nil
}
