package risk

import (
	"context"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
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

// TestStagePersistsScore verifies the computed score and its derived level
// are persisted onto the session, so they survive into JSON/YAML reports and
// machine-readable events.
func TestStagePersistsScore(t *testing.T) {
	s := models.NewSession()
	s.AddFinding(&models.Finding{
		Title:      "critical",
		Severity:   models.SeverityCritical,
		Confidence: models.ConfidenceConfirmed,
		Status:     models.StatusConfirmed,
	})
	env := &core.Env{Session: s}
	st := &Stage{}
	if err := st.Run(context.Background(), env); err != nil {
		t.Fatalf("Stage.Run: %v", err)
	}
	if s.RiskScore != MaxScore {
		t.Errorf("Session.RiskScore = %d, want %d", s.RiskScore, MaxScore)
	}
	if s.RiskLevel != "critical" {
		t.Errorf("Session.RiskLevel = %q, want critical", s.RiskLevel)
	}
}

func TestLevelBuckets(t *testing.T) {
	for score, want := range map[int]string{
		0:   "none",
		10:  "low",
		40:  "medium",
		60:  "high",
		100: "critical",
	} {
		if got := Level(score); got != want {
			t.Errorf("Level(%d) = %q, want %q", score, got, want)
		}
	}
}
