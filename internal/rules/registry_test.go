package rules

import (
	"context"
	"testing"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// countingRule is a test rule that records how many times it was evaluated.
type countingRule struct {
	id    string
	calls int
}

func (r *countingRule) ID() string                { return "TEST-001" }
func (r *countingRule) Name() string              { return "test rule" }
func (r *countingRule) Category() string          { return "test" }
func (r *countingRule) Description() string       { return "test rule for registry" }
func (r *countingRule) Severity() models.Severity { return models.SeverityMedium }
func (r *countingRule) MitreRefs() []string       { return nil }
func (r *countingRule) Evaluate(ctx context.Context, ec EvaluationContext) ([]models.Finding, error) {
	r.calls++
	return []models.Finding{
		{
			ID:          models.NewID("fnd"),
			Title:       "test finding",
			Category:    "test",
			Severity:    models.SeverityMedium,
			Confidence:  models.ConfidenceConfirmed,
			Status:      models.StatusDetected,
			Description: "produced by counting rule",
		},
	}, nil
}

func TestRegistryEvaluate(t *testing.T) {
	reg := NewRegistry()
	rule := &countingRule{}
	if err := reg.Register(rule); err != nil {
		t.Fatalf("Register: %v", err)
	}

	findings := reg.Evaluate(context.Background(), EvaluationContext{})
	if len(findings) != 1 {
		t.Fatalf("Evaluate returned %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.RuleID != "TEST-001" {
		t.Errorf("RuleID = %q, want TEST-001", f.RuleID)
	}
	if f.ID == "" {
		t.Error("finding ID is empty; registry should assign one")
	}
	if f.Timestamp.IsZero() {
		t.Error("finding timestamp is zero; registry should set it")
	}
}

func TestRegistryDuplicateRejected(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&countingRule{}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := reg.Register(&countingRule{}); err == nil {
		t.Error("duplicate registration should fail")
	}
}

func TestRegistryErrorTolerated(t *testing.T) {
	reg := NewRegistry()
	good := &countingRule{}
	bad := &failingRule{}
	if err := reg.Register(good); err != nil {
		t.Fatalf("Register good: %v", err)
	}
	if err := reg.Register(bad); err != nil {
		t.Fatalf("Register bad: %v", err)
	}

	findings := reg.Evaluate(context.Background(), EvaluationContext{})
	// One good finding plus one diagnostic for the failing rule.
	if len(findings) != 2 {
		t.Fatalf("Evaluate returned %d findings, want 2 (1 real + 1 diagnostic)", len(findings))
	}
	diagnostic := findings[1]
	if diagnostic.Status != models.StatusInformational {
		t.Errorf("diagnostic status = %q, want informational", diagnostic.Status)
	}
}

type failingRule struct{ countingRule }

func (r *failingRule) ID() string { return "TEST-BAD" }

func (r *failingRule) Evaluate(ctx context.Context, ec EvaluationContext) ([]models.Finding, error) {
	return nil, &evaluationFailure{}
}

type evaluationFailure struct{}

func (e *evaluationFailure) Error() string { return "boom" }
