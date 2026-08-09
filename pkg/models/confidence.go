package models

// Confidence expresses how sure the framework is that a finding is real as
// opposed to a false positive. It is tracked independently from Severity so
// that a high-impact issue with weak evidence is reported differently from a
// high-impact issue that has been validated.
type Confidence string

const (
	// ConfidenceConfirmed means the condition was validated with direct
	// evidence (for example, a configuration value or an observed behavior).
	ConfidenceConfirmed Confidence = "confirmed"
	// ConfidenceHigh means the evidence strongly supports the finding.
	ConfidenceHigh Confidence = "high"
	// ConfidenceMedium means the evidence is suggestive but not conclusive.
	ConfidenceMedium Confidence = "medium"
	// ConfidenceLow means the finding is a lead that needs validation.
	ConfidenceLow Confidence = "low"
)

// ParseConfidence converts a case-insensitive string into a Confidence,
// defaulting to Low for unknown input.
func ParseConfidence(s string) Confidence {
	switch Confidence(s) {
	case ConfidenceConfirmed, ConfidenceHigh, ConfidenceMedium, ConfidenceLow:
		return Confidence(s)
	default:
		return ConfidenceLow
	}
}

// Valid reports whether c is a recognized confidence value.
func (c Confidence) Valid() bool {
	return c == ConfidenceConfirmed || c == ConfidenceHigh ||
		c == ConfidenceMedium || c == ConfidenceLow
}
