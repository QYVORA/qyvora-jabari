package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Session ties one assessment run to a target and its collected findings.
// Sessions make assessments resumable, comparable, and automatable (for
// example "jabari diff session-A session-B").
type Session struct {
	ID       string        `json:"id"`
	TargetID string        `json:"target_id"`
	Profile  string        `json:"profile"`
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end,omitempty"`
	Stages   []string      `json:"stages,omitempty"`
	Findings []*Finding    `json:"findings,omitempty"`
	Apps     []Application `json:"apps,omitempty"`
	Errors   []string      `json:"errors,omitempty"`
	// Pocs records every proof-of-concept module execution performed against
	// the session's findings. It is empty unless the poc stage ran.
	Pocs []*PocRun `json:"pocs,omitempty"`
	// RiskScore is the 0..100 target risk figure computed by the risk stage.
	// It is informational and never a substitute for reading the findings.
	RiskScore int `json:"risk_score,omitempty"`
	// RiskLevel is the derived severity label (low, medium, high, critical)
	// for the overall target risk score.
	RiskLevel string `json:"risk_level,omitempty"`
	OutputDir string `json:"output_dir,omitempty"`
}

// NewSession creates a session with a fresh identifier and the start time
// recorded. It does not attach a target; callers set TargetID explicitly.
func NewSession() *Session {
	return &Session{
		ID:       NewID("sess"),
		Start:    time.Now().UTC(),
		Stages:   []string{},
		Findings: []*Finding{},
		Errors:   []string{},
	}
}

// AddFinding records a finding against the session.
func (s *Session) AddFinding(f *Finding) {
	f.SessionID = s.ID
	s.Findings = append(s.Findings, f)
}

// AddPoc records a proof-of-concept run against the session.
func (s *Session) AddPoc(run *PocRun) {
	s.Pocs = append(s.Pocs, run)
}

// Finish marks the session end time and records the ordered stage list.
func (s *Session) Finish() {
	s.End = time.Now().UTC()
}

// NewID returns a random lowercase hex identifier with the given prefix.
// It is used for targets, sessions, evidence, and runs so that identifiers
// are unique without coordination.
func NewID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is unrecoverable in practice; fall back to a
		// time-based suffix so we never emit an empty identifier.
		return prefix + "-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return prefix + "-" + hex.EncodeToString(b)
}
