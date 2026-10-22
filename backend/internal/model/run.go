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
