package models

// RiskLevel is the Part B safety rating for an offensive validation module.
// It governs how much blast radius a module may have and whether it requires
// explicit escalation flags.
type RiskLevel string

const (
	// RiskS1 is reversible and low impact (for example reading a manifest).
	RiskS1 RiskLevel = "S1"
	// RiskS2 is reversible with limited impact (for example starting an
	// exported activity in a controlled way).
	RiskS2 RiskLevel = "S2"
	// RiskS3 is higher impact and requires --allow-risky (for example
	// installing a repackaged test APK).
	RiskS3 RiskLevel = "S3"
	// RiskS4 requires --allow-risky and is disabled in compliance and
	// read-only profiles (for example msfvenom passthrough).
	RiskS4 RiskLevel = "S4"
)

// Rank returns the numeric rank of a risk level (S1=1 .. S4=4). Unknown
// levels rank 0 and are treated as "no rating".
func (r RiskLevel) Rank() int {
	switch r {
	case RiskS1:
		return 1
	case RiskS2:
		return 2
	case RiskS3:
		return 3
	case RiskS4:
		return 4
	default:
		return 0
	}
}

// RequiresRiskyFlag reports whether a module at this level is gated behind
// the explicit --allow-risky escalation.
func (r RiskLevel) RequiresRiskyFlag() bool {
	return r == RiskS3 || r == RiskS4
}
