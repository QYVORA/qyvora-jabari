package risk

import (
	"testing"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

func TestScoreEmpty(t *testing.T) {
	if got := Score(nil); got != 0 {
		t.Errorf("Score(nil) = %d, want 0", got)
	}
}

func TestScoreConfirmedCritical(t *testing.T) {
	findings := []*models.Finding{
		{
			Title:      "critical",
			Severity:   models.SeverityCritical,
			Confidence: models.ConfidenceConfirmed,
			Status:     models.StatusConfirmed,
		},
	}
	if got := Score(findings); got != MaxScore {
		t.Errorf("Score = %d, want %d", got, MaxScore)
	}
}

func TestScoreDetectedContributesLess(t *testing.T) {
	makeF := func(sev models.Severity, conf models.Confidence) *models.Finding {
		return &models.Finding{Severity: sev, Confidence: conf, Status: models.StatusDetected}
	}
	confirmed := Score([]*models.Finding{
		makeF(models.SeverityHigh, models.ConfidenceConfirmed),
	})
	detected := Score([]*models.Finding{
		makeF(models.SeverityHigh, models.ConfidenceHigh),
	})
	if detected >= confirmed {
		t.Errorf("detected score %d should be below confirmed score %d", detected, confirmed)
	}
}

func TestScoreExcludesFalsePositives(t *testing.T) {
	findings := []*models.Finding{
		{
			Severity:   models.SeverityCritical,
			Confidence: models.ConfidenceConfirmed,
			Status:     models.StatusFalsePositive,
		},
	}
	if got := Score(findings); got != 0 {
		t.Errorf("Score = %d, want 0 for false positive", got)
	}
}

func TestSeverityCounts(t *testing.T) {
	var c models.SeverityCounts
	c.Critical = 1
	c.Medium = 3
	if c.Tally() != 4 {
		t.Errorf("Tally = %d, want 4", c.Tally())
	}
}
