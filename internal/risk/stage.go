// Package risk implements severity and confidence scoring for findings and
// the overall target risk assessment.
package risk

import (
	"context"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// MaxScore is the total attainable risk score for a target. Scoring is
// relative so reports stay comparable across targets.
const MaxScore = 100

// Stage computes per-finding and overall risk scores after validation.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "risk" }

// Run attaches a weighted risk score to the session based on the confirmed
// findings. The score and its derived level are persisted on the session so
// they survive into JSON, YAML, reports, and machine-readable events. Scores
// are informational; they are never a substitute for reading the findings
// themselves.
func (s *Stage) Run(_ context.Context, env *core.Env) error {
	if env.Session == nil {
		return nil
	}
	score := Score(env.Session.Findings)
	env.Session.RiskScore = score
	env.Session.RiskLevel = Level(score)
	if env.Log != nil {
		env.Log.Info("target risk score: %d/%d (%s)", score, MaxScore, env.Session.RiskLevel)
	}
	return nil
}

// Level maps a 0..100 risk score to a severity label. The buckets match the
// severity vocabulary used across the report formats.
func Level(score int) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 35:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "none"
	}
}

// Score computes a 0..MaxScore risk figure from findings. Each finding
// contributes severity weight scaled by confidence; confirmed critical
// findings dominate the score.
func Score(findings []*models.Finding) int {
	var total float64
	var maxWeight float64
	for _, f := range findings {
		if f == nil || f.Status == models.StatusFalsePositive || f.Status == models.StatusResolved {
			continue
		}
		w := severityWeight(f.Severity)
		c := confidenceWeight(f.Confidence)
		total += float64(w) * c
		maxWeight += float64(maxSeverityWeight) * confidenceWeight(models.ConfidenceConfirmed)
	}
	if maxWeight == 0 {
		return 0
	}
	score := int(float64(MaxScore) * total / maxWeight)
	if score > MaxScore {
		score = MaxScore
	}
	return score
}

const maxSeverityWeight = 4

func severityWeight(s models.Severity) int {
	return s.Weights()
}

// confidenceWeight scales severity by how sure we are the finding is real.
// Detected-but-unconfirmed findings deliberately contribute less.
func confidenceWeight(c models.Confidence) float64 {
	switch c {
	case models.ConfidenceConfirmed:
		return 1.0
	case models.ConfidenceHigh:
		return 0.8
	case models.ConfidenceMedium:
		return 0.6
	case models.ConfidenceLow:
		return 0.3
	default:
		return 0.5
	}
}
