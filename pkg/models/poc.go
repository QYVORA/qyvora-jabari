package models

import "time"

// PocStatus is the outcome of one proof-of-concept module execution.
type PocStatus string

const (
	// PocProven means the PoC successfully demonstrated the weakness on the
	// live target. The corresponding finding is marked exploited.
	PocProven PocStatus = "proven"
	// PocNotProven means the PoC ran but could not demonstrate the weakness
	// on the live target.
	PocNotProven PocStatus = "not-proven"
	// PocSkipped means the PoC was considered but not executed (for example
	// a high-risk module that was not explicitly allowed).
	PocSkipped PocStatus = "skipped"
	// PocError means the PoC failed for a technical reason.
	PocError PocStatus = "error"
)

// PocRun is the structured record of one PoC module execution against a
// finding. Runs are appended to the session so reports and the JSONL event
// stream carry the proof that exploitation was (or was not) demonstrated.
type PocRun struct {
	ID         string    `json:"id"`
	Module     string    `json:"module"`
	FindingID  string    `json:"finding_id,omitempty"`
	Status     PocStatus `json:"status"`
	Risk       string    `json:"risk,omitempty"`
	Summary    string    `json:"summary,omitempty"`
	Error      string    `json:"error,omitempty"`
	Evidence   []string  `json:"evidence,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Proven reports whether the run demonstrated the weakness.
func (r *PocRun) Proven() bool { return r != nil && r.Status == PocProven }
