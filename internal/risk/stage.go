// Package risk implements severity and confidence scoring for findings and
// the overall target risk assessment.
package risk

import (
	"context"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// MaxScore is the total attainable risk score for a target. Scoring is
// relative so reports stay comparable across targets.
const MaxScore = 100

// Stage computes per-finding and overall risk scores after validation.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "risk" }

// Run attaches a weighted risk score to the session based on the confirmed
// findings. Scores are informational; they are never a substitute for reading
// the findings themselves.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Session == nil {
		return nil
	}
	score := Score(env.Session.Findings)
	if env.Log != nil {
		env.Log.Info("target risk score: %d/%d", score, MaxScore)
	}
	return nil
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
