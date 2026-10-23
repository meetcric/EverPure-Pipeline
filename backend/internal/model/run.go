package model

import "time"

// Status is where a run is in its lifecycle.
type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusFlaky   Status = "flaky"
)

// Run is one execution of a pipeline.
// FinishedAt and DurationMs are pointers: nil until the run ends, and omitted from JSON.
type Run struct {
	ID          string     `json:"id"`
	PipelineID  string     `json:"pipelineId"`
	Status      Status     `json:"status"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	DurationMs  *int64     `json:"durationMs,omitempty"`
	TriggeredBy string     `json:"triggeredBy"`
	CommitSha   string     `json:"commitSha"`
}

// Valid reports whether s is one of the five known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusQueued, StatusRunning, StatusPassed, StatusFailed, StatusFlaky:
		return true
	}
	return false
}

// Terminal reports whether the run has finished (passed, failed or flaky).
func (s Status) Terminal() bool {
	return s == StatusPassed || s == StatusFailed || s == StatusFlaky
}

// Stage orders statuses along the lifecycle: queued 0, running 1, finished 2.
// A run only ever moves to a higher stage, so a lower or equal one is a stale event.
func (s Status) Stage() int {
	switch {
	case s == StatusQueued:
		return 0
	case s == StatusRunning:
		return 1
	default:
		return 2
	}
}

// CreateRunInput is what a client may send on POST /pipelines/{id}/runs.
// No ID, status or times: a new run always starts as queued, timed by the DB.
type CreateRunInput struct {
	TriggeredBy string `json:"triggeredBy"`
	CommitSha   string `json:"commitSha"`
}

// UpdateRunInput is what a client may send on PATCH /runs/{id}.
type UpdateRunInput struct {
	Status Status `json:"status"`
}

// RunWithNotes is the GET /runs/{id} response. Embedding Run puts its fields
// at the top level of the JSON, next to "notes".
type RunWithNotes struct {
	Run
	Notes []Note `json:"notes"`
}

// ListRunsInput is the query string of GET /pipelines/{id}/runs, as the client sent it.
// Zero values mean "not sent": no status filter, default limit, first page.
type ListRunsInput struct {
	Status Status
	Limit  int
	Cursor string
}

// RunCursor is the position of the last run on a page. The next page starts strictly after it.
type RunCursor struct {
	StartedAt time.Time
	ID        string
}

// ListRunsQuery is the validated query the store runs. After is nil on the first page.
type ListRunsQuery struct {
	PipelineID string
	Status     Status
	Limit      int
	After      *RunCursor
}

// RunPage is one page of runs. NextCursor is null when there are no more pages.
type RunPage struct {
	Items      []Run   `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
