package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
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
