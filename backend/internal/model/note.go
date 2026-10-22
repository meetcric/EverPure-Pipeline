package model

import "time"

// Note is a triage comment attached to a run.
type Note struct {
	ID        string    `json:"id"`
	RunID     string    `json:"runId"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}
