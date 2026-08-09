// Package models defines the normalized data model shared by every stage of
// the JABARI assessment pipeline.
//
// The model intentionally uses plain structs with JSON tags so that findings,
// targets, and sessions can be serialized for evidence, reporting, and
// machine-to-machine integration without coupling to any particular stage.
package models

import "strings"

// Severity describes the business or security impact of a finding.
type Severity string

const (
	SeverityCritical      Severity = "critical"
	SeverityHigh          Severity = "high"
	SeverityMedium        Severity = "medium"
	SeverityLow           Severity = "low"
	SeverityInformational Severity = "informational"
)

// ParseSeverity converts a case-insensitive string into a Severity. It falls
// back to Informational for unknown input so that unparsed data degrades
// safely instead of producing a zero-value severity of "".
func ParseSeverity(s string) Severity {
	norm := Severity(strings.ToLower(s))
	switch norm {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInformational:
		return norm
	default:
		return SeverityInformational
	}
}

// Weights returns a numeric weight used to sort findings by impact. Higher
// is more severe. Unknown severities collapse to Informational (weight 0).
func (s Severity) Weights() int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// Valid reports whether s is a recognized severity value.
func (s Severity) Valid() bool {
	return s.Weights() != 0 || s == SeverityInformational
}
