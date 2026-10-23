package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"pipelinepulse/internal/model"
)

// NoteStore implements service.NoteStore.
type NoteStore struct {
	pool *pgxpool.Pool
}

func NewNoteStore(pool *pgxpool.Pool) *NoteStore {
	return &NoteStore{pool: pool}
}

// Create inserts a note. created_at comes from the column default (now()).
func (s *NoteStore) Create(ctx context.Context, n model.Note) (model.Note, error) {
	rows, err := s.pool.Query(ctx,
		`INSERT INTO notes (id, run_id, author, body) VALUES ($1, $2, $3, $4)
		 RETURNING id, run_id, author, body, created_at`,
		n.ID, n.RunID, n.Author, n.Body)
	if err != nil {
		return model.Note{}, fmt.Errorf("create note: %w", err)
	}
	note, err := pgx.CollectOneRow(rows, scanNote)
	// The run doesn't exist: the FK rejected the row.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == fkViolation {
		return model.Note{}, model.ErrNotFound
	}
	if err != nil {
		return model.Note{}, fmt.Errorf("create note: %w", err)
	}
	return note, nil
}

func (s *NoteStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanNote(row pgx.CollectableRow) (model.Note, error) {
	var n model.Note
	err := row.Scan(&n.ID, &n.RunID, &n.Author, &n.Body, &n.CreatedAt)
	return n, err
}
