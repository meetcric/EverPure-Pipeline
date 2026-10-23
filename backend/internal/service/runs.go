package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"pipelinepulse/internal/model"
)

// RunStore is what the run service needs from the DB layer.
type RunStore interface {
	Get(ctx context.Context, id string) (model.Run, error)
	Create(ctx context.Context, r model.Run) (model.Run, error)
	UpdateStatus(ctx context.Context, id string, to model.Status, from []model.Status) (model.Run, error)
	ListNotes(ctx context.Context, runID string) ([]model.Note, error)
	List(ctx context.Context, q model.ListRunsQuery) ([]model.Run, error)
	PipelineExists(ctx context.Context, id string) (bool, error)
}

// RunService implements httpapi.RunService.
type RunService struct {
	store RunStore
}

func NewRunService(store RunStore) *RunService {
	return &RunService{store: store}
}

// Get returns the run with its notes, oldest note first.
func (s *RunService) Get(ctx context.Context, id string) (model.RunWithNotes, error) {
	run, err := s.store.Get(ctx, id)
	if err != nil {
		return model.RunWithNotes{}, err
	}
	notes, err := s.store.ListNotes(ctx, id)
	if err != nil {
		return model.RunWithNotes{}, err
	}
	return model.RunWithNotes{Run: run, Notes: notes}, nil
}

const (
	defaultRunLimit = 20
	maxRunLimit     = 100
)

// List returns one page of a pipeline's runs, newest first.
func (s *RunService) List(ctx context.Context, pipelineID string, in model.ListRunsInput) (model.RunPage, error) {
	if in.Status != "" && !in.Status.Valid() {
		return model.RunPage{}, &model.ValidationError{Message: "status must be one of queued, running, passed, failed, flaky"}
	}
	limit := in.Limit
	if limit == 0 {
		limit = defaultRunLimit
	}
	if limit < 1 || limit > maxRunLimit {
		return model.RunPage{}, &model.ValidationError{Message: "limit must be between 1 and 100"}
	}
	q := model.ListRunsQuery{PipelineID: pipelineID, Status: in.Status, Limit: limit + 1}
	if in.Cursor != "" {
		after, err := decodeCursor(in.Cursor)
		if err != nil {
			return model.RunPage{}, err
		}
		q.After = &after
	}

	// We asked for one extra row: if it came back, there is another page.
	runs, err := s.store.List(ctx, q)
	if err != nil {
		return model.RunPage{}, err
	}
	page := model.RunPage{Items: runs}
	if len(runs) > limit {
		page.Items = runs[:limit]
		last := page.Items[limit-1]
		next := encodeCursor(model.RunCursor{StartedAt: last.StartedAt, ID: last.ID})
		page.NextCursor = &next
	}

	// An empty page is fine for a real pipeline with no runs, but a missing pipeline is a 404.
	if len(runs) == 0 {
		ok, err := s.store.PipelineExists(ctx, pipelineID)
		if err != nil {
			return model.RunPage{}, err
		}
		if !ok {
			return model.RunPage{}, model.ErrNotFound
		}
	}
	return page, nil
}

// encodeCursor makes an opaque token: base64 of "startedAt|id". Clients send it back
// unchanged, so we can change what's inside later without breaking them.
func encodeCursor(c model.RunCursor) string {
	raw := c.StartedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (model.RunCursor, error) {
	invalid := &model.ValidationError{Message: "cursor is invalid"}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return model.RunCursor{}, invalid
	}
	// The timestamp never contains "|", so the first one separates it from the id.
	ts, id, ok := strings.Cut(string(raw), "|")
	if !ok || id == "" {
		return model.RunCursor{}, invalid
	}
	startedAt, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return model.RunCursor{}, invalid
	}
	return model.RunCursor{StartedAt: startedAt, ID: id}, nil
}

// Create validates the input and saves a queued run. The store returns ErrNotFound
// if the pipeline does not exist.
func (s *RunService) Create(ctx context.Context, pipelineID string, in model.CreateRunInput) (model.Run, error) {
	triggeredBy := strings.TrimSpace(in.TriggeredBy)
	if triggeredBy == "" {
		return model.Run{}, &model.ValidationError{Message: "triggeredBy is required"}
	}
	if utf8.RuneCountInString(triggeredBy) > maxNameLen {
		return model.Run{}, &model.ValidationError{Message: "triggeredBy must be at most 100 characters"}
	}
	sha := strings.ToLower(strings.TrimSpace(in.CommitSha))
	if !commitShaRe.MatchString(sha) {
		return model.Run{}, &model.ValidationError{Message: "commitSha must be 4 to 64 hex characters"}
	}
	return s.store.Create(ctx, model.Run{
		ID:          uuid.NewString(),
		PipelineID:  pipelineID,
		TriggeredBy: triggeredBy,
		CommitSha:   sha,
	})
}

// commitShaRe accepts a short or full git SHA: git abbreviates to at least 4 characters,
// a SHA-1 is 40 and a SHA-256 is 64.
var commitShaRe = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// allowedFrom lists, for each target status, the statuses a run may move from.
// queued is never a target, and passed/failed/flaky are never a source (terminal).
var allowedFrom = map[model.Status][]model.Status{
	model.StatusRunning: {model.StatusQueued},
	model.StatusPassed:  {model.StatusRunning},
	model.StatusFlaky:   {model.StatusRunning},
	model.StatusFailed:  {model.StatusQueued, model.StatusRunning},
}

// UpdateStatus moves the run to in.Status if its current status allows it.
// The check and the write are one SQL statement, so two concurrent PATCHes
// can't both move the run out of the same status.
func (s *RunService) UpdateStatus(ctx context.Context, id string, in model.UpdateRunInput) (model.Run, error) {
	if !in.Status.Valid() {
		return model.Run{}, &model.ValidationError{Message: "status must be one of queued, running, passed, failed, flaky"}
	}
	run, err := s.store.UpdateStatus(ctx, id, in.Status, allowedFrom[in.Status])
	if !errors.Is(err, model.ErrNotFound) {
		return run, err
	}
	// Nothing was updated: either the run doesn't exist, or its status doesn't allow this move.
	cur, err := s.store.Get(ctx, id)
	if err != nil {
		return model.Run{}, err
	}
	if cur.Status == in.Status {
		return cur, nil // already there: a no-op, so retrying a PATCH is safe
	}
	return model.Run{}, fmt.Errorf("%w: %s → %s", model.ErrInvalidTransition, cur.Status, in.Status)
}
