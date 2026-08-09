package models

import "time"

// FindingStatus tracks where a finding sits in the assessment lifecycle.
// Validation moves findings from Detected to Confirmed (or marks them as a
// false positive). See the exploitability ladder in the Part B spec for how
// these map onto offense validation stages.
type FindingStatus string

const (
	// StatusDetected means a rule fired but no independent validation has
	// been performed. This is the default for every new finding.
	StatusDetected FindingStatus = "detected"
	// StatusConfirmed means validation established the condition is real.
	StatusConfirmed FindingStatus = "confirmed"
	// StatusFalsePositive means validation showed the condition is not real.
	StatusFalsePositive FindingStatus = "false-positive"
	// StatusResolved means a previously valid finding no longer applies.
	StatusResolved FindingStatus = "resolved"
	// StatusInformational means the finding is context rather than a defect.
	StatusInformational FindingStatus = "informational"
)

// ParseFindingStatus converts a case-insensitive string into a
// FindingStatus, defaulting to Detected for unknown input.
func ParseFindingStatus(s string) FindingStatus {
	switch FindingStatus(s) {
	case StatusDetected, StatusConfirmed, StatusFalsePositive, StatusResolved, StatusInformational:
		return FindingStatus(s)
	default:
		return StatusDetected
	}
}

// Finding is the normalized representation of a security condition discovered
// during an assessment.
//
// Every finding carries the evidence that supports it plus enough context to
// reproduce the result later. Findings are the unit that flows from the rule
// engine through validation, risk scoring, and reporting.
type Finding struct {
	ID             string            `json:"id"`
	TargetID       string            `json:"target_id"`
	SessionID      string            `json:"session_id"`
	Title          string            `json:"title"`
	Category       string            `json:"category"`
	Description    string            `json:"description"`
	Impact         string            `json:"impact,omitempty"`
	Recommendation string            `json:"recommendation,omitempty"`
	Severity       Severity          `json:"severity"`
	Confidence     Confidence        `json:"confidence"`
	Status         FindingStatus     `json:"status"`
	RuleID         string            `json:"rule_id,omitempty"`
	Evidence       []Evidence        `json:"evidence,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	References     []string          `json:"references,omitempty"`
	// MitreRefs maps MITRE ATT&CK for Mobile technique IDs (for example
	// T1404) to this finding when a rule is annotated with them.
	MitreRefs []string `json:"mitre_refs,omitempty"`
	// CVEs lists CVE identifiers correlated with the condition when the
	// vulnerability intelligence database is available.
	CVEs []string `json:"cves,omitempty"`
	// Exploitability carries the Part B ladder position
	// (detected|static|dynamic|exploited|chained) once exploitation
	// validation exists. It is empty for findings that were not exercised.
	Exploitability string    `json:"exploitability,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// AddEvidence appends an evidence item to the finding and returns the
// finding for chaining. Duplicate references to the same artifact are not
// deduplicated here; callers may use Evidence.Hash for that.
func (f *Finding) AddEvidence(ev Evidence) *Finding {
	f.Evidence = append(f.Evidence, ev)
	return f
}

// SeverityCounts summarizes how many findings exist at each severity. It is
// the shape used by reports and the terminal summary.
type SeverityCounts struct {
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	Informational int `json:"informational"`
}

// Tally returns the total number of findings represented by the counts.
func (c SeverityCounts) Tally() int {
	return c.Critical + c.High + c.Medium + c.Low + c.Informational
}
